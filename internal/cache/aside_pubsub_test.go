package cache

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"git.itopcms.com/astrueus/doc/internal/cache/cachetest"
	"git.itopcms.com/astrueus/doc/internal/cache/store"
	"git.itopcms.com/astrueus/doc/internal/config"
)

func TestAsidePubSubFlushesPeerL1(t *testing.T) {
	bus := NewMemoryBus()
	l2 := store.NewMemory()
	l1a := store.NewMemory()
	l1b := store.NewMemory()
	a := NewAside[int](l1a, l2, WithBus(bus), WithOrigin("inst-a"))
	b := NewAside[int](l1b, l2, WithBus(bus), WithOrigin("inst-b"))
	t.Cleanup(a.Close)
	t.Cleanup(b.Close)

	ctx := context.Background()
	opt := Options{TTL: time.Minute, L1TTL: time.Minute}
	if err := a.Set(ctx, "k", 1, opt); err != nil {
		t.Fatal(err)
	}
	v, err := b.GetOrLoad(ctx, "k", opt, func(context.Context) (int, error) {
		t.Fatal("应命中共享 L2")
		return 0, nil
	})
	if err != nil || v != 1 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	if _, ok, _ := l1b.Get(ctx, "k"); !ok {
		t.Fatal("B 的 L1 应已回填")
	}

	if err := a.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := l1b.Get(ctx, "k"); ok {
		t.Fatal("A Delete 后 B 的 L1 应被 Pub/Sub 清掉")
	}
}

func TestAsidePubSubSkipsSelf(t *testing.T) {
	bus := NewMemoryBus()
	l1, l2 := cachetest.MemoryPair()
	a := NewAside[int](l1, l2, WithBus(bus), WithOrigin("only"))
	t.Cleanup(a.Close)
	ctx := context.Background()
	opt := Options{TTL: time.Minute}
	if err := a.Set(ctx, "k", 1, opt); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	// 自己发的消息不应再走一遍 handler 误伤；Delete 已本地删除，这里只确认不 panic。
	if _, ok, _ := l1.Get(ctx, "k"); ok {
		t.Fatal("本地 Delete 后 L1 应空")
	}
}

func TestSettingsFromDefaults(t *testing.T) {
	s := SettingsFrom(config.CacheSection{
		RedisHost: "127.0.0.1:6380",
	})
	if s.Mode != ModeLocal {
		t.Fatalf("mode=%s", s.Mode)
	}
	if s.RedisAddr != "127.0.0.1:6380" {
		t.Fatalf("addr=%s", s.RedisAddr)
	}
	if s.AsidePrefix != DefaultKeyPrefix {
		t.Fatalf("prefix=%s", s.AsidePrefix)
	}
	if s.PubSubChannel != DefaultPubSubChannel {
		t.Fatalf("channel=%s", s.PubSubChannel)
	}
}

func TestSettingsFromRedisAddrWins(t *testing.T) {
	s := SettingsFrom(config.CacheSection{
		RedisHost: "host:1",
		RedisAddr: "addr:2",
		Mode:      "CHAIN",
	})
	if s.Mode != ModeChain {
		t.Fatalf("mode=%s", s.Mode)
	}
	if s.RedisAddr != "addr:2" {
		t.Fatalf("addr=%s", s.RedisAddr)
	}
}

func TestOpenLocalGetOrLoad(t *testing.T) {
	rt, err := Open(context.Background(), Settings{
		Mode:          ModeLocal,
		L1MaxCost:     1 << 20,
		L1NumCounters: 1_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.Close() })
	a := NewAsideFrom[int](rt)
	t.Cleanup(a.Close)
	var loads atomic.Int32
	v, err := a.GetOrLoad(context.Background(), "k", Options{TTL: time.Minute}, func(context.Context) (int, error) {
		loads.Add(1)
		return 42, nil
	})
	if err != nil || v != 42 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	v, err = a.GetOrLoad(context.Background(), "k", Options{TTL: time.Minute}, func(context.Context) (int, error) {
		loads.Add(1)
		return 0, nil
	})
	if err != nil || v != 42 || loads.Load() != 1 {
		t.Fatalf("v=%d loads=%d err=%v", v, loads.Load(), err)
	}
}

func TestOpenUnknownMode(t *testing.T) {
	_, err := Open(context.Background(), Settings{Mode: "disk"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenRedisRequiresAddr(t *testing.T) {
	_, err := Open(context.Background(), Settings{Mode: ModeRedis})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMetricsSnapshotMap(t *testing.T) {
	m := &Metrics{}
	m.L1Hit.Add(2)
	m.Miss.Add(1)
	got := m.Snapshot().Map()
	if got["cache_l1_hit"] != 2 || got["cache_miss"] != 1 {
		t.Fatalf("%v", got)
	}
}
