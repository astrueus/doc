package cache

import (
	"context"
	"sync"
)

// InvalidateMsg 为跨实例 L1 失效通知。
type InvalidateMsg struct {
	Op     string   `json:"op"`
	Keys   []string `json:"keys"`
	Origin string   `json:"origin"`
}

// Broadcaster 把失效事件发给其他进程。
type Broadcaster interface {
	Publish(ctx context.Context, msg InvalidateMsg) error
	Subscribe(ctx context.Context, origin string, handler func(InvalidateMsg)) error
}

type noopBus struct{}

func (noopBus) Publish(context.Context, InvalidateMsg) error { return nil }

func (noopBus) Subscribe(context.Context, string, func(InvalidateMsg)) error {
	return nil
}

// NewNoopBus 不广播（单实例 / 未配 Redis）。
func NewNoopBus() Broadcaster { return noopBus{} }

// MemoryBus 进程内总线，用于单测模拟双实例。
type MemoryBus struct {
	mu   sync.Mutex
	subs map[string]func(InvalidateMsg)
}

// NewMemoryBus 返回可多订阅者的内存总线。
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{subs: make(map[string]func(InvalidateMsg))}
}

func (b *MemoryBus) Publish(_ context.Context, msg InvalidateMsg) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	handlers := make([]func(InvalidateMsg), 0, len(b.subs))
	for origin, h := range b.subs {
		if origin == msg.Origin {
			continue
		}
		handlers = append(handlers, h)
	}
	b.mu.Unlock()
	for _, h := range handlers {
		h(msg)
	}
	return nil
}

func (b *MemoryBus) Subscribe(ctx context.Context, origin string, handler func(InvalidateMsg)) error {
	if b == nil || origin == "" || handler == nil {
		return nil
	}
	b.mu.Lock()
	b.subs[origin] = handler
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subs, origin)
		b.mu.Unlock()
	}()
	return nil
}
