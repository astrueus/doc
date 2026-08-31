package cache

import (
	"context"
	"sync"
	"time"
)

// TagIndex 维护 tag → keys，供 InvalidateTag。
type TagIndex interface {
	Add(ctx context.Context, key string, tags []string, ttl time.Duration) error
	KeysOf(ctx context.Context, tags []string) ([]string, error)
	RemoveKeys(ctx context.Context, keys []string) error
	RemoveTags(ctx context.Context, tags []string) error
}

type memoryTags struct {
	mu sync.Mutex
	m  map[string]map[string]struct{}
}

// NewMemoryTags 返回进程内 tag 索引（mode=local / 单测）。
func NewMemoryTags() TagIndex {
	return &memoryTags{m: make(map[string]map[string]struct{})}
}

func (t *memoryTags) Add(_ context.Context, key string, tags []string, _ time.Duration) error {
	if t == nil || key == "" || len(tags) == 0 {
		return nil
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
	return nil
}

func (t *memoryTags) KeysOf(_ context.Context, tags []string) ([]string, error) {
	if t == nil {
		return nil, nil
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
	return out, nil
}

func (t *memoryTags) RemoveKeys(_ context.Context, keys []string) error {
	if t == nil || len(keys) == 0 {
		return nil
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
	return nil
}

func (t *memoryTags) RemoveTags(_ context.Context, tags []string) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, tag := range tags {
		delete(t.m, tag)
	}
	return nil
}
