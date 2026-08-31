package store

import (
	"context"
	"sync"
	"time"
)

type memItem struct {
	val []byte
	exp time.Time // 零值表示不过期
}

// Memory 进程内 Store，供单测。并发安全。
type Memory struct {
	mu  sync.RWMutex
	m   map[string]memItem
	now func() time.Time
}

// NewMemory 返回空的内存 Store。
func NewMemory() *Memory {
	return &Memory{
		m:   make(map[string]memItem),
		now: time.Now,
	}
}

// Get 读取未过期的值。
func (s *Memory) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	item, ok := s.m[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !item.exp.IsZero() && !s.now().Before(item.exp) {
		s.mu.Lock()
		delete(s.m, key)
		s.mu.Unlock()
		return nil, false, nil
	}
	out := make([]byte, len(item.val))
	copy(out, item.val)
	return out, true, nil
}

// Set 写入并按 ttl 过期。ttl<=0 表示不过期。
func (s *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	item := memItem{val: append([]byte(nil), value...)}
	if ttl > 0 {
		item.exp = s.now().Add(ttl)
	}
	s.mu.Lock()
	s.m[key] = item
	s.mu.Unlock()
	return nil
}

// Del 删除若干 key；不存在的 key 忽略。
func (s *Memory) Del(_ context.Context, keys ...string) error {
	s.mu.Lock()
	for _, k := range keys {
		delete(s.m, k)
	}
	s.mu.Unlock()
	return nil
}

// Len 返回当前条目数（含已过期未清理的）。仅测试用。
func (s *Memory) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
