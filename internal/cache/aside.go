package cache

import (
	"context"
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

type tagIndex struct {
	mu sync.Mutex
	m  map[string]map[string]struct{}
}

func newTagIndex() *tagIndex {
	return &tagIndex{m: make(map[string]map[string]struct{})}
}

func (t *tagIndex) add(key string, tags []string) {
	if t == nil || key == "" || len(tags) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		set, ok := t.m[tag]
		if !ok {
			set = make(map[string]struct{})
			t.m[tag] = set
		}
		set[key] = struct{}{}
	}
}

func (t *tagIndex) keysOf(tags []string) []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for _, tag := range tags {
		for k := range t.m[tag] {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	return out
}

func (t *tagIndex) removeKeys(keys []string) {
	if t == nil || len(keys) == 0 {
		return
	}
	drop := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		drop[k] = struct{}{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for tag, set := range t.m {
		for k := range set {
			if _, ok := drop[k]; ok {
				delete(set, k)
			}
		}
		if len(set) == 0 {
			delete(t.m, tag)
		}
	}
}

func (t *tagIndex) removeTags(tags []string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, tag := range tags {
		delete(t.m, tag)
	}
}

// Aside 是 Cache-Aside Facade：L1 → L2 → singleflight(loader)。
type Aside[T any] struct {
	l1      store.Store
	l2      store.Store
	codec   codec.Codec
	loadSF  Coalesce
	softSF  Coalesce
	metrics *Metrics
	now     func() time.Time
	tags    *tagIndex

	refreshWG sync.WaitGroup
}

// AsideOption 配置 Aside。
type AsideOption func(*asideConfig)

type asideConfig struct {
	codec   codec.Codec
	metrics *Metrics
	now     func() time.Time
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

// NewAside 构造 Facade。l1 必填；l2 可为 nil（仅本地）。
func NewAside[T any](l1, l2 store.Store, opts ...AsideOption) *Aside[T] {
	cfg := asideConfig{
		codec:   codec.Msgpack(),
		metrics: &Metrics{},
		now:     time.Now,
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
	return &Aside[T]{
		l1:      l1,
		l2:      l2,
		codec:   cfg.codec,
		metrics: cfg.metrics,
		now:     cfg.now,
		tags:    newTagIndex(),
	}
}

// Metrics 返回计数器。
func (a *Aside[T]) Metrics() *Metrics {
	if a == nil {
		return nil
	}
	return a.metrics
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
	opt = opt.normalized()

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
	return a.writeValue(ctx, key, v, opt.normalized())
}

// Delete 删除 L1/L2 上的 key。
func (a *Aside[T]) Delete(ctx context.Context, keys ...string) error {
	if a == nil {
		return nil
	}
	return a.deleteKeys(ctx, keys...)
}

// InvalidateTag 按 tag 失效（T12-a 为进程内索引；T12-b 迁 Redis SET）。
func (a *Aside[T]) InvalidateTag(ctx context.Context, tags ...string) error {
	if a == nil || len(tags) == 0 {
		return nil
	}
	keys := a.tags.keysOf(tags)
	if err := a.deleteKeys(ctx, keys...); err != nil {
		return err
	}
	a.tags.removeTags(tags)
	return nil
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
	a.tags.add(key, opt.Tags)
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
	a.tags.removeKeys(keys)
	return first
}
