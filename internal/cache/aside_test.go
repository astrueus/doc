package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache/cachetest"
	"git.itopcms.com/astrueus/doc/internal/cache/store"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func TestAsideStampedeSingleLoad(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	var loads atomic.Int32
	var gate sync.WaitGroup
	gate.Add(1)
	load := func(context.Context) (int, error) {
		gate.Wait()
		loads.Add(1)
		return 7, nil
	}
	opt := Options{TTL: time.Minute, SoftTTL: 50 * time.Second}

	const n = 64
	var start, done sync.WaitGroup
	start.Add(n)
	done.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			defer done.Done()
			start.Done()
			start.Wait()
			v, err := a.GetOrLoad(context.Background(), "hot", opt, load)
			if err != nil || v != 7 {
				errCh <- fmt.Errorf("v=%d err=%v", v, err)
			}
		}()
	}
	start.Wait()
	gate.Done()
	done.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("并发 miss 回源次数=%d，期望 1", got)
	}
	if snap := a.Metrics().Snapshot(); snap.Load != 1 {
		t.Fatalf("metrics load=%d", snap.Load)
	}
}

func TestAsideNegativeCache(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		loads.Add(1)
		return 0, ErrNotFound
	}
	opt := Options{TTL: time.Minute, CacheNull: true, NullTTL: time.Minute}

	_, err := a.GetOrLoad(context.Background(), "missing", opt, load)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	_, err = a.GetOrLoad(context.Background(), "missing", opt, load)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("second err=%v", err)
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("负缓存后仍回源 %d 次", got)
	}
	if snap := a.Metrics().Snapshot(); snap.NullHit < 1 {
		t.Fatalf("expected NullHit, got %+v", snap)
	}
}

func TestAsideNegativeCacheDisabled(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		loads.Add(1)
		return 0, ErrNotFound
	}
	opt := Options{TTL: time.Minute, CacheNull: false}

	_, _ = a.GetOrLoad(context.Background(), "missing", opt, load)
	_, _ = a.GetOrLoad(context.Background(), "missing", opt, load)
	if got := loads.Load(); got != 2 {
		t.Fatalf("未开负缓存应每次回源，got=%d", got)
	}
}

func TestAsideSoftRefresh(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	a := NewAside[string](l1, l2, WithClock(clk.Now))
	var loads atomic.Int32
	load := func(context.Context) (string, error) {
		n := loads.Add(1)
		return fmt.Sprintf("v%d", n), nil
	}
	opt := Options{TTL: 10 * time.Second, SoftTTL: 5 * time.Second, L1TTL: time.Minute}

	v, err := a.GetOrLoad(context.Background(), "doc", opt, load)
	if err != nil || v != "v1" {
		t.Fatalf("first: %q %v", v, err)
	}

	clk.Advance(6 * time.Second)
	v, err = a.GetOrLoad(context.Background(), "doc", opt, load)
	if err != nil || v != "v1" {
		t.Fatalf("stale: %q %v", v, err)
	}
	a.waitRefresh()
	if got := loads.Load(); got != 2 {
		t.Fatalf("Soft 刷新未回源，loads=%d", got)
	}

	v, err = a.GetOrLoad(context.Background(), "doc", opt, load)
	if err != nil || v != "v2" {
		t.Fatalf("after refresh: %q %v", v, err)
	}
	if got := loads.Load(); got != 2 {
		t.Fatalf("刷新后不应再回源，loads=%d", got)
	}
	if snap := a.Metrics().Snapshot(); snap.SoftRefresh < 1 {
		t.Fatalf("expected SoftRefresh, got %+v", snap)
	}
}

func TestAsideLoaderErrorNotCached(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		loads.Add(1)
		return 0, io.ErrUnexpectedEOF
	}
	opt := Options{TTL: time.Minute}

	_, err := a.GetOrLoad(context.Background(), "k", opt, load)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err=%v", err)
	}
	_, err = a.GetOrLoad(context.Background(), "k", opt, load)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("second err=%v", err)
	}
	if got := loads.Load(); got != 2 {
		t.Fatalf("临时错误不应缓存，loads=%d", got)
	}
}

func TestAsideL2FillsL1(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		loads.Add(1)
		return 9, nil
	}
	opt := Options{TTL: time.Minute, L1TTL: time.Minute}

	if _, err := a.GetOrLoad(context.Background(), "k", opt, load); err != nil {
		t.Fatal(err)
	}
	if err := l1.Del(context.Background(), "k"); err != nil {
		t.Fatal(err)
	}

	v, err := a.GetOrLoad(context.Background(), "k", opt, load)
	if err != nil || v != 9 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("L2 命中仍回源 loads=%d", got)
	}
	if _, ok, err := l1.Get(context.Background(), "k"); err != nil || !ok {
		t.Fatalf("应回填 L1 ok=%v err=%v", ok, err)
	}
}

func TestAsideInvalidateTag(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	opt := Options{TTL: time.Minute, Tags: []string{"book:1"}}
	ctx := context.Background()
	if err := a.Set(ctx, "a", 1, opt); err != nil {
		t.Fatal(err)
	}
	if err := a.Set(ctx, "b", 2, opt); err != nil {
		t.Fatal(err)
	}
	if err := a.InvalidateTag(ctx, "book:1"); err != nil {
		t.Fatal(err)
	}

	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		loads.Add(1)
		return 3, nil
	}
	if _, err := a.GetOrLoad(ctx, "a", opt, load); err != nil {
		t.Fatal(err)
	}
	if loads.Load() != 1 {
		t.Fatalf("tag 失效后应回源，loads=%d", loads.Load())
	}
}

func TestAsideHardExpireReloads(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	a := NewAside[int](l1, l2, WithClock(clk.Now))
	var loads atomic.Int32
	load := func(context.Context) (int, error) {
		return int(loads.Add(1)), nil
	}
	opt := Options{TTL: 5 * time.Second, L1TTL: time.Minute}

	v, err := a.GetOrLoad(context.Background(), "k", opt, load)
	if err != nil || v != 1 {
		t.Fatalf("first: %d %v", v, err)
	}
	clk.Advance(6 * time.Second)
	v, err = a.GetOrLoad(context.Background(), "k", opt, load)
	if err != nil || v != 2 {
		t.Fatalf("hard expire: %d %v", v, err)
	}
}

func TestAsideDelete(t *testing.T) {
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2)
	opt := Options{TTL: time.Minute}
	ctx := context.Background()
	if err := a.Set(ctx, "k", 1, opt); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	var loads atomic.Int32
	_, _ = a.GetOrLoad(ctx, "k", opt, func(context.Context) (int, error) {
		loads.Add(1)
		return 2, nil
	})
	if loads.Load() != 1 {
		t.Fatal("Delete 后应回源")
	}
}

func TestJitteredTTL(t *testing.T) {
	if got := jitteredTTL(time.Second, 0); got != time.Second {
		t.Fatalf("jitter=0: %s", got)
	}
	base := 10 * time.Second
	seen := map[time.Duration]struct{}{}
	for i := 0; i < 80; i++ {
		got := jitteredTTL(base, 0.1)
		if got < 9*time.Second || got > 11*time.Second {
			t.Fatalf("jitter 超出 ±10%%: %s", got)
		}
		seen[got] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatal("jitter 未打散 TTL")
	}
}

func TestKeyBuilder(t *testing.T) {
	k := Keys()
	if k.DocumentByID(3) != "doc:v1:document:id:3" {
		t.Fatal(k.DocumentByID(3))
	}
	if k.DocumentByIdentify(2, "intro") != "doc:v1:document:book:2:ident:intro" {
		t.Fatal(k.DocumentByIdentify(2, "intro"))
	}
	if k.TagBook(8) != "book:8" {
		t.Fatal(k.TagBook(8))
	}
	if k.TagBlog(4) != "blog:4" {
		t.Fatal(k.TagBlog(4))
	}
}

func TestMemoryStoreExpiry(t *testing.T) {
	s := store.NewMemory()
	ctx := context.Background()
	if err := s.Set(ctx, "k", []byte("v"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "k"); !ok {
		t.Fatal("should exist")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("should expire")
	}
}
