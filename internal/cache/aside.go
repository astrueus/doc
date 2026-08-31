package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache/codec"
	"git.itopcms.com/astrueus/doc/internal/cache/store"
)

// Loader 在缓存未命中时回源。返回 ErrNotFound 表示确认不存在。
type Loader[T any] func(ctx context.Context) (T, error)

type envelope struct {
	Payload  []byte `msgpack:"p"`
	Null     bool   `msgpack:"n"`
	StoredAt int64  `msgpack:"s"`
	SoftAt   int64  `msgpack:"o"`
	HardAt   int64  `msgpack:"h"`
}

// Aside 是 Cache-Aside Facade：L1 → L2 → singleflight(loader)。
type Aside[T any] struct {
	l1            store.Store
	l2            store.Store
	codec         codec.Codec
	loadSF        Coalesce
	softSF        Coalesce
	metrics       *Metrics
	now           func() time.Time
	tags          TagIndex
	bus           Broadcaster
	origin        string
	defaultJitter float64
	subCancel     context.CancelFunc

	refreshWG sync.WaitGroup
}

// AsideOption 配置 Aside。
type AsideOption func(*asideConfig)

type asideConfig struct {
	codec         codec.Codec
	metrics       *Metrics
	now           func() time.Time
	tags          TagIndex
	bus           Broadcaster
	origin        string
	defaultJitter float64
	skipSubscribe bool
}

// WithCodec 覆盖默认 msgpack。
func WithCodec(c codec.Codec) AsideOption {
	return func(cfg *asideConfig) { cfg.codec = c }
}

// WithMetrics 注入计数器；nil 则内建一份。
func WithMetrics(m *Metrics) AsideOption {
	return func(cfg *asideConfig) { cfg.metrics = m }
}

// WithClock 注入时钟，便于单测 Soft-TTL。
func WithClock(now func() time.Time) AsideOption {
	return func(cfg *asideConfig) { cfg.now = now }
}

// WithTagIndex 注入 tag 索引；nil 则用进程内实现。
func WithTagIndex(t TagIndex) AsideOption {
	return func(cfg *asideConfig) { cfg.tags = t }
}

// WithBus 注入失效广播；nil 则不广播。
func WithBus(b Broadcaster) AsideOption {
	return func(cfg *asideConfig) { cfg.bus = b }
}

// WithOrigin 设置本实例 ID，Pub/Sub 用来跳过自己。
func WithOrigin(id string) AsideOption {
	return func(cfg *asideConfig) { cfg.origin = id }
}

// WithDefaultJitter 在 Options.Jitter 为 0 时使用的抖动比例。
func WithDefaultJitter(j float64) AsideOption {
	return func(cfg *asideConfig) { cfg.defaultJitter = j }
}

func withSkipSubscribe() AsideOption {
	return func(cfg *asideConfig) { cfg.skipSubscribe = true }
}

// NewAside 构造 Facade。l1 必填；l2 可为 nil（仅本地）。
func NewAside[T any](l1, l2 store.Store, opts ...AsideOption) *Aside[T] {
	cfg := asideConfig{
		codec:   codec.Msgpack(),
		metrics: &Metrics{},
		now:     time.Now,
		tags:    NewMemoryTags(),
		bus:     NewNoopBus(),
		origin:  newOriginID(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.codec == nil {
		cfg.codec = codec.Msgpack()
	}
	if cfg.metrics == nil {
		cfg.metrics = &Metrics{}
	}
	if cfg.now == nil {
		cfg.now = time.Now
	}
	if cfg.tags == nil {
		cfg.tags = NewMemoryTags()
	}
	if cfg.bus == nil {
		cfg.bus = NewNoopBus()
	}
	if cfg.origin == "" {
		cfg.origin = newOriginID()
	}
	a := &Aside[T]{
		l1:            l1,
		l2:            l2,
		codec:         cfg.codec,
		metrics:       cfg.metrics,
		now:           cfg.now,
		tags:          cfg.tags,
		bus:           cfg.bus,
		origin:        cfg.origin,
		defaultJitter: cfg.defaultJitter,
	}
	if !cfg.skipSubscribe {
		ctx, cancel := context.WithCancel(context.Background())
		a.subCancel = cancel
		_ = a.bus.Subscribe(ctx, a.origin, a.onRemoteInvalidate)
	}
	return a
}

// NewAsideFrom 从 Runtime 构造 Facade（订阅由 Runtime 统一处理）。
func NewAsideFrom[T any](rt *Runtime, opts ...AsideOption) *Aside[T] {
	if rt == nil {
		return NewAside[T](nil, nil, opts...)
	}
	base := []AsideOption{
		WithTagIndex(rt.Tags),
		WithBus(rt.Bus),
		WithOrigin(rt.Origin),
		WithMetrics(rt.Metrics),
		WithDefaultJitter(rt.DefaultJitter),
		withSkipSubscribe(),
	}
	return NewAside[T](rt.L1, rt.L2, append(base, opts...)...)
}

func newOriginID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "local"
	}
	return hex.EncodeToString(b[:])
}

// Close 停止 Pub/Sub 订阅并等待 Soft 刷新结束。
func (a *Aside[T]) Close() {
	if a == nil {
		return
	}
	if a.subCancel != nil {
		a.subCancel()
		a.subCancel = nil
	}
	a.waitRefresh()
}

func (a *Aside[T]) onRemoteInvalidate(msg InvalidateMsg) {
	if a == nil || a.l1 == nil || len(msg.Keys) == 0 {
		return
	}
	_ = a.l1.Del(context.Background(), msg.Keys...)
}

// Metrics 返回计数器。
func (a *Aside[T]) Metrics() *Metrics {
	if a == nil {
		return nil
	}
	return a.metrics
}

func (a *Aside[T]) applyJitter(opt Options) Options {
	if opt.Jitter == 0 && a.defaultJitter > 0 {
		opt.Jitter = a.defaultJitter
	}
	return opt
}

// GetOrLoad 按 L1 → L2 → loader 读取；Soft-TTL 命中时返回旧值并异步刷新。
func (a *Aside[T]) GetOrLoad(ctx context.Context, key string, opt Options, load Loader[T]) (T, error) {
	var zero T
	if a == nil || a.l1 == nil {
		return zero, errors.New("cache: aside not initialized")
	}
	if load == nil {
		return zero, errors.New("cache: loader required")
	}
	opt = a.applyJitter(opt.normalized())

	if v, ok, err := a.readLayers(ctx, key, opt, load); ok {
		return v, err
	}

	got, err, shared := a.loadSF.Do(key, func() (any, error) {
		if v, hit, rerr := a.readLayers(ctx, key, opt, nil); hit {
			if rerr != nil {
				return zero, rerr
			}
			return v, nil
		}
		a.metrics.Miss.Add(1)
		return a.loadAndStore(ctx, key, opt, load)
	})
	if shared {
		a.metrics.LoadShared.Add(1)
	}
	if err != nil {
		return zero, err
	}
	v, _ := got.(T)
	return v, nil
}

// Set 显式写入正缓存。
func (a *Aside[T]) Set(ctx context.Context, key string, v T, opt Options) error {
	if a == nil || a.l1 == nil {
		return errors.New("cache: aside not initialized")
	}
	return a.writeValue(ctx, key, v, a.applyJitter(opt.normalized()))
}

// Delete 删除 L1/L2 上的 key，并广播让对端只刷 L1。
func (a *Aside[T]) Delete(ctx context.Context, keys ...string) error {
	if a == nil {
		return nil
	}
	return a.deleteKeys(ctx, keys...)
}

// InvalidateTag 按 tag 失效：查出 keys → Delete → 删 tagset。
func (a *Aside[T]) InvalidateTag(ctx context.Context, tags ...string) error {
	if a == nil || len(tags) == 0 || a.tags == nil {
		return nil
	}
	keys, err := a.tags.KeysOf(ctx, tags)
	if err != nil {
		return err
	}
	if err := a.deleteKeys(ctx, keys...); err != nil {
		return err
	}
	return a.tags.RemoveTags(ctx, tags)
}

func (a *Aside[T]) waitRefresh() {
	a.refreshWG.Wait()
}

func (a *Aside[T]) readLayers(ctx context.Context, key string, opt Options, load Loader[T]) (T, bool, error) {
	var zero T
	now := a.now()

	if rec, ok, err := a.getEnvelope(ctx, a.l1, key); err != nil {
		return zero, false, err
	} else if ok {
		v, hit, rerr := a.serveRecord(ctx, key, rec, opt, now, load)
		if hit {
			a.metrics.L1Hit.Add(1)
		}
		return v, hit, rerr
	}

	if a.l2 == nil {
		return zero, false, nil
	}
	rec, ok, err := a.getEnvelope(ctx, a.l2, key)
	if err != nil || !ok {
		return zero, false, err
	}
	v, hit, rerr := a.serveRecord(ctx, key, rec, opt, now, load)
	if hit {
		a.metrics.L2Hit.Add(1)
		a.fillL1(ctx, key, rec, opt)
	}
	return v, hit, rerr
}

func (a *Aside[T]) serveRecord(ctx context.Context, key string, rec envelope, opt Options, now time.Time, load Loader[T]) (T, bool, error) {
	var zero T
	if rec.HardAt != 0 && hardExpired(now, time.Unix(0, rec.HardAt)) {
		_ = a.deleteKeys(ctx, key)
		return zero, false, nil
	}
	if rec.Null {
		a.metrics.NullHit.Add(1)
		return zero, true, ErrNotFound
	}
	var v T
	if err := a.codec.Unmarshal(rec.Payload, &v); err != nil {
		_ = a.deleteKeys(ctx, key)
		return zero, false, nil
	}
	if load != nil && rec.SoftAt != 0 && softExpired(now, time.Unix(0, rec.SoftAt)) {
		a.spawnSoftRefresh(ctx, key, opt, load)
	}
	return v, true, nil
}

func (a *Aside[T]) spawnSoftRefresh(ctx context.Context, key string, opt Options, load Loader[T]) {
	a.metrics.SoftRefresh.Add(1)
	refreshCtx := context.WithoutCancel(ctx)
	a.refreshWG.Add(1)
	go func() {
		defer a.refreshWG.Done()
		_, _, _ = a.softSF.Do(key, func() (any, error) {
			_, err := a.loadAndStore(refreshCtx, key, opt, load)
			return nil, err
		})
	}()
}

func (a *Aside[T]) loadAndStore(ctx context.Context, key string, opt Options, load Loader[T]) (T, error) {
	var zero T
	a.metrics.Load.Add(1)
	val, err := load(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			if opt.CacheNull {
				if serr := a.writeNull(ctx, key, opt); serr != nil {
					return zero, errors.Join(err, serr)
				}
			}
			return zero, err
		}
		a.metrics.LoadErr.Add(1)
		return zero, err
	}
	if serr := a.writeValue(ctx, key, val, opt); serr != nil {
		return val, serr
	}
	return val, nil
}

func (a *Aside[T]) getEnvelope(ctx context.Context, st store.Store, key string) (envelope, bool, error) {
	var rec envelope
	if st == nil {
		return rec, false, nil
	}
	raw, ok, err := st.Get(ctx, key)
	if err != nil || !ok {
		return rec, false, err
	}
	if err := a.codec.Unmarshal(raw, &rec); err != nil {
		_ = st.Del(ctx, key)
		return rec, false, nil
	}
	return rec, true, nil
}

func (a *Aside[T]) fillL1(ctx context.Context, key string, rec envelope, opt Options) {
	raw, err := a.codec.Marshal(rec)
	if err != nil {
		return
	}
	ttl := opt.L1TTL
	if rec.HardAt != 0 {
		remain := time.Unix(0, rec.HardAt).Sub(a.now())
		if remain <= 0 {
			return
		}
		ttl = opt.l1TTLCapped(remain)
	}
	if ttl > 0 {
		_ = a.l1.Set(ctx, key, raw, ttl)
	}
}

func (a *Aside[T]) writeValue(ctx context.Context, key string, v T, opt Options) error {
	payload, err := a.codec.Marshal(v)
	if err != nil {
		return err
	}
	return a.writeEnvelope(ctx, key, envelope{Payload: payload, Null: false}, opt, opt.TTL)
}

func (a *Aside[T]) writeNull(ctx context.Context, key string, opt Options) error {
	return a.writeEnvelope(ctx, key, envelope{Null: true}, opt, opt.NullTTL)
}

func (a *Aside[T]) writeEnvelope(ctx context.Context, key string, rec envelope, opt Options, baseTTL time.Duration) error {
	now := a.now()
	hardTTL := jitteredTTL(baseTTL, opt.Jitter)
	if hardTTL < 0 {
		hardTTL = 0
	}
	rec.StoredAt = now.UnixNano()
	if opt.SoftTTL > 0 && !rec.Null {
		rec.SoftAt = now.Add(opt.SoftTTL).UnixNano()
	}
	if hardTTL > 0 {
		rec.HardAt = now.Add(hardTTL).UnixNano()
	}
	raw, err := a.codec.Marshal(rec)
	if err != nil {
		return err
	}
	l1ttl := opt.l1TTLCapped(hardTTL)
	if l1ttl <= 0 && hardTTL > 0 {
		l1ttl = hardTTL
	}
	if err := a.l1.Set(ctx, key, raw, l1ttl); err != nil {
		return err
	}
	if a.l2 != nil && hardTTL > 0 {
		if err := a.l2.Set(ctx, key, raw, hardTTL); err != nil {
			return err
		}
	}
	if a.tags != nil && len(opt.Tags) > 0 {
		if err := a.tags.Add(ctx, key, opt.Tags, hardTTL); err != nil {
			return err
		}
	}
	return nil
}

func (a *Aside[T]) deleteKeys(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	var first error
	if err := a.l1.Del(ctx, keys...); err != nil {
		first = err
	}
	if a.l2 != nil {
		if err := a.l2.Del(ctx, keys...); err != nil && first == nil {
			first = err
		}
	}
	if a.tags != nil {
		if err := a.tags.RemoveKeys(ctx, keys); err != nil && first == nil {
			first = err
		}
	}
	if a.bus != nil {
		if err := a.bus.Publish(ctx, InvalidateMsg{Op: "del", Keys: append([]string(nil), keys...), Origin: a.origin}); err != nil && first == nil {
			first = err
		}
	}
	return first
}
