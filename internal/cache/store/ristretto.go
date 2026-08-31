package store

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// Ristretto 为进程内 L1。
type Ristretto struct {
	c *ristretto.Cache[string, []byte]
}

// RistrettoConfig L1 容量。
type RistrettoConfig struct {
	NumCounters int64
	MaxCost     int64
}

// NewRistretto 构造 L1。cfg 零值用文档默认（约 64MiB / 1e6 counters）。
func NewRistretto(cfg RistrettoConfig) (*Ristretto, error) {
	if cfg.NumCounters <= 0 {
		cfg.NumCounters = 1_000_000
	}
	if cfg.MaxCost <= 0 {
		cfg.MaxCost = 64 << 20
	}
	c, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: cfg.NumCounters,
		MaxCost:     cfg.MaxCost,
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("ristretto: %w", err)
	}
	return &Ristretto{c: c}, nil
}

func (s *Ristretto) Get(_ context.Context, key string) ([]byte, bool, error) {
	if s == nil || s.c == nil {
		return nil, false, nil
	}
	v, ok := s.c.Get(key)
	if !ok {
		return nil, false, nil
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, true, nil
}

func (s *Ristretto) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.c == nil {
		return nil
	}
	cost := int64(len(value))
	if cost < 1 {
		cost = 1
	}
	buf := append([]byte(nil), value...)
	if ttl > 0 {
		s.c.SetWithTTL(key, buf, cost, ttl)
	} else {
		s.c.Set(key, buf, cost)
	}
	s.c.Wait()
	return nil
}

func (s *Ristretto) Del(_ context.Context, keys ...string) error {
	if s == nil || s.c == nil {
		return nil
	}
	for _, k := range keys {
		s.c.Del(k)
	}
	return nil
}

// Close 释放 L1。
func (s *Ristretto) Close() {
	if s != nil && s.c != nil {
		s.c.Close()
	}
}
