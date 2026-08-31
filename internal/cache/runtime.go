package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"git.itopcms.com/astrueus/doc/internal/cache/store"
	"git.itopcms.com/astrueus/doc/internal/config"
)

const (
	// ModeLocal 仅进程内 L1（Ristretto），无跨实例。
	ModeLocal = "local"
	// ModeRedis 单层 Redis 当 L1，带 Tag 与 Pub/Sub。
	ModeRedis = "redis"
	// ModeChain L1 Ristretto + L2 Redis + Tag + Pub/Sub。
	ModeChain = "chain"

	// DefaultPubSubChannel 为跨实例失效频道。
	DefaultPubSubChannel = "doc:cache:invalidate"
	defaultL1MaxCost     = 64 << 20
	defaultL1NumCounters = 1_000_000
)

// Settings 为 Open 所需的 Aside 拓扑配置（与现网 beego cache_provider 独立）。
type Settings struct {
	Mode          string
	L1MaxCost     int64
	L1NumCounters int64
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	PubSubChannel string
	DefaultJitter float64
	AsidePrefix   string
}

// SettingsFrom 从 CacheSection 抽出 T12 新键；空 Mode 视为 local。
func SettingsFrom(sec config.CacheSection) Settings {
	mode := strings.ToLower(strings.TrimSpace(sec.Mode))
	addr := strings.TrimSpace(sec.RedisAddr)
	if addr == "" {
		addr = strings.TrimSpace(sec.RedisHost)
	}
	return Settings{
		Mode:          mode,
		L1MaxCost:     sec.L1MaxCost,
		L1NumCounters: sec.L1NumCounters,
		RedisAddr:     addr,
		RedisPassword: sec.RedisPassword,
		RedisDB:       sec.RedisDB,
		PubSubChannel: sec.PubSubChannel,
		DefaultJitter: sec.DefaultJitter,
		AsidePrefix:   sec.AsidePrefix,
	}.normalized()
}

func (s Settings) normalized() Settings {
	s.Mode = strings.ToLower(strings.TrimSpace(s.Mode))
	if s.Mode == "" {
		s.Mode = ModeLocal
	}
	if s.L1MaxCost <= 0 {
		s.L1MaxCost = defaultL1MaxCost
	}
	if s.L1NumCounters <= 0 {
		s.L1NumCounters = defaultL1NumCounters
	}
	if strings.TrimSpace(s.PubSubChannel) == "" {
		s.PubSubChannel = DefaultPubSubChannel
	}
	if s.AsidePrefix == "" {
		s.AsidePrefix = DefaultKeyPrefix
	}
	if s.DefaultJitter < 0 {
		s.DefaultJitter = 0
	}
	return s
}

// Runtime 是按 mode 组好的 L1/L2/Tag/Bus。T12-b 不接入 RegisterCache。
type Runtime struct {
	L1            store.Store
	L2            store.Store
	Tags          TagIndex
	Bus           Broadcaster
	Metrics       *Metrics
	Keys          KeyBuilder
	Origin        string
	DefaultJitter float64
	Mode          string
	rdb           *redis.Client
	cancel        context.CancelFunc
}

// Open 按 Settings 组装 Runtime。local 不连 Redis；redis/chain 会 Ping。
func Open(ctx context.Context, cfg Settings) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg = cfg.normalized()
	subCtx, cancel := context.WithCancel(context.Background())
	rt := &Runtime{
		Metrics:       &Metrics{},
		Origin:        newOriginID(),
		Keys:          KeyBuilder{Prefix: cfg.AsidePrefix},
		DefaultJitter: cfg.DefaultJitter,
		Mode:          cfg.Mode,
		cancel:        cancel,
	}

	switch cfg.Mode {
	case ModeLocal:
		l1, err := store.NewRistretto(store.RistrettoConfig{
			NumCounters: cfg.L1NumCounters,
			MaxCost:     cfg.L1MaxCost,
		})
		if err != nil {
			cancel()
			return nil, err
		}
		rt.L1 = l1
		rt.Tags = NewMemoryTags()
		rt.Bus = NewNoopBus()
	case ModeRedis:
		rdb, err := dialRedis(ctx, cfg)
		if err != nil {
			cancel()
			return nil, err
		}
		rt.rdb = rdb
		rt.L1 = store.NewRedis(rdb)
		rt.Tags = NewRedisTags(rdb, cfg.AsidePrefix)
		rt.Bus = NewRedisBus(rdb, cfg.PubSubChannel)
	case ModeChain:
		l1, err := store.NewRistretto(store.RistrettoConfig{
			NumCounters: cfg.L1NumCounters,
			MaxCost:     cfg.L1MaxCost,
		})
		if err != nil {
			cancel()
			return nil, err
		}
		rdb, err := dialRedis(ctx, cfg)
		if err != nil {
			l1.Close()
			cancel()
			return nil, err
		}
		rt.rdb = rdb
		rt.L1 = l1
		rt.L2 = store.NewRedis(rdb)
		rt.Tags = NewRedisTags(rdb, cfg.AsidePrefix)
		rt.Bus = NewRedisBus(rdb, cfg.PubSubChannel)
	default:
		cancel()
		return nil, fmt.Errorf("cache: unknown mode %q (want local|redis|chain)", cfg.Mode)
	}

	if err := rt.Bus.Subscribe(subCtx, rt.Origin, func(msg InvalidateMsg) {
		if rt.L1 == nil || len(msg.Keys) == 0 {
			return
		}
		_ = rt.L1.Del(context.Background(), msg.Keys...)
	}); err != nil {
		_ = rt.Close()
		return nil, err
	}
	return rt, nil
}

func dialRedis(ctx context.Context, cfg Settings) (*redis.Client, error) {
	if strings.TrimSpace(cfg.RedisAddr) == "" {
		return nil, errors.New("cache: redis addr required")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("cache redis ping: %w", err)
	}
	return rdb, nil
}

// Close 停止订阅并释放 L1 / Redis 客户端。
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if c, ok := r.L1.(interface{ Close() }); ok {
		c.Close()
	}
	if r.rdb != nil {
		err := r.rdb.Close()
		r.rdb = nil
		return err
	}
	return nil
}
