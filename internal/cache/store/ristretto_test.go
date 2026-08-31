package store

import (
	"context"
	"testing"
	"time"
)

func TestRistrettoRoundTrip(t *testing.T) {
	s, err := NewRistretto(RistrettoConfig{NumCounters: 1_000, MaxCost: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	ctx := context.Background()
	if err := s.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Get(ctx, "k")
	if err != nil || !ok || string(got) != "v" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
	if err := s.Del(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ = s.Get(ctx, "k"); ok {
		t.Fatal("deleted key still present")
	}
}

func TestRistrettoTTL(t *testing.T) {
	s, err := NewRistretto(RistrettoConfig{NumCounters: 1_000, MaxCost: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	ctx := context.Background()
	if err := s.Set(ctx, "k", []byte("v"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, ok, _ := s.Get(ctx, "k"); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("ristretto TTL 未过期")
}
