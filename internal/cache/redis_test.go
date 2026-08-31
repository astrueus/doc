package cache

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"git.itopcms.com/astrueus/doc/internal/cache/store"
)

func testMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	return mr, rdb
}

func TestRedisStoreRoundTrip(t *testing.T) {
	mr, rdb := testMiniRedis(t)
	st := store.NewRedis(rdb)
	ctx := context.Background()
	if err := st.Set(ctx, "doc:v1:k", []byte("v"), time.Second); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.Get(ctx, "doc:v1:k")
	if err != nil || !ok || string(got) != "v" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
	mr.FastForward(2 * time.Second)
	if _, ok, err = st.Get(ctx, "doc:v1:k"); err != nil || ok {
		t.Fatalf("expired ok=%v err=%v", ok, err)
	}
	if err := st.Set(ctx, "a", []byte("1"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := st.Del(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ = st.Get(ctx, "a"); ok {
		t.Fatal("deleted")
	}
}

func TestRedisTagsIndex(t *testing.T) {
	_, rdb := testMiniRedis(t)
	tags := NewRedisTags(rdb, DefaultKeyPrefix)
	ctx := context.Background()
	if err := tags.Add(ctx, "doc:v1:a", []string{"book:1"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := tags.Add(ctx, "doc:v1:b", []string{"book:1", "document:2"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	keys, err := tags.KeysOf(ctx, []string{"book:1"})
	if err != nil || len(keys) != 2 {
		t.Fatalf("keys=%v err=%v", keys, err)
	}
	if err := tags.RemoveTags(ctx, []string{"book:1"}); err != nil {
		t.Fatal(err)
	}
	keys, err = tags.KeysOf(ctx, []string{"book:1"})
	if err != nil || len(keys) != 0 {
		t.Fatalf("after del keys=%v err=%v", keys, err)
	}
}

func TestRedisBusSkipsOrigin(t *testing.T) {
	_, rdb := testMiniRedis(t)
	bus := NewRedisBus(rdb, "doc:test:invalidate")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var n atomic.Int32
	if err := bus.Subscribe(ctx, "peer-b", func(InvalidateMsg) { n.Add(1) }); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(ctx, InvalidateMsg{Op: "del", Keys: []string{"k"}, Origin: "peer-b"}); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(ctx, InvalidateMsg{Op: "del", Keys: []string{"k"}, Origin: "peer-a"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if n.Load() == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("got %d messages, want 1", n.Load())
}

func TestOpenChainPubSubFlushesPeerL1(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := Settings{
		Mode:          ModeChain,
		RedisAddr:     mr.Addr(),
		L1MaxCost:     1 << 20,
		L1NumCounters: 1_000,
		PubSubChannel: "doc:test:chain",
		DefaultJitter: 0,
	}
	ctx := context.Background()
	aRT, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	bRT, err := Open(ctx, cfg)
	if err != nil {
		_ = aRT.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = aRT.Close()
		_ = bRT.Close()
	})
	a := NewAsideFrom[int](aRT)
	b := NewAsideFrom[int](bRT)
	t.Cleanup(a.Close)
	t.Cleanup(b.Close)

	opt := Options{TTL: time.Minute, L1TTL: time.Minute}
	if err := a.Set(ctx, "k", 7, opt); err != nil {
		t.Fatal(err)
	}
	v, err := b.GetOrLoad(ctx, "k", opt, func(context.Context) (int, error) {
		t.Fatal("应命中 L2")
		return 0, nil
	})
	if err != nil || v != 7 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	if _, ok, _ := bRT.L1.Get(ctx, "k"); !ok {
		t.Fatal("B L1 应回填")
	}
	if err := a.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok, _ := bRT.L1.Get(ctx, "k"); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("chain 模式下 A Delete 后 B 的 L1 未被刷掉")
}

func TestAsideRedisTagInvalidate(t *testing.T) {
	_, rdb := testMiniRedis(t)
	l1 := store.NewMemory()
	l2 := store.NewRedis(rdb)
	tags := NewRedisTags(rdb, DefaultKeyPrefix)
	a := NewAside[int](l1, l2, WithTagIndex(tags), WithOrigin("t"))
	t.Cleanup(a.Close)
	ctx := context.Background()
	opt := Options{TTL: time.Minute, Tags: []string{"book:9"}}
	if err := a.Set(ctx, "doc:v1:x", 1, opt); err != nil {
		t.Fatal(err)
	}
	if err := a.InvalidateTag(ctx, "book:9"); err != nil {
		t.Fatal(err)
	}
	var loads atomic.Int32
	_, _ = a.GetOrLoad(ctx, "doc:v1:x", opt, func(context.Context) (int, error) {
		loads.Add(1)
		return 2, nil
	})
	if loads.Load() != 1 {
		t.Fatalf("tag 失效后应回源, loads=%d", loads.Load())
	}
}
