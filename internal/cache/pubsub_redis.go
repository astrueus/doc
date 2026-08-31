package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type redisBus struct {
	rdb     *redis.Client
	channel string
}

// NewRedisBus 用 Redis Pub/Sub 刷其他实例的 L1。
func NewRedisBus(rdb *redis.Client, channel string) Broadcaster {
	if channel == "" {
		channel = "doc:cache:invalidate"
	}
	return &redisBus{rdb: rdb, channel: channel}
}

func (b *redisBus) Publish(ctx context.Context, msg InvalidateMsg) error {
	if b == nil || b.rdb == nil || len(msg.Keys) == 0 {
		return nil
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("cache pubsub marshal: %w", err)
	}
	return b.rdb.Publish(ctx, b.channel, raw).Err()
}

func (b *redisBus) Subscribe(ctx context.Context, origin string, handler func(InvalidateMsg)) error {
	if b == nil || b.rdb == nil || handler == nil {
		return nil
	}
	sub := b.rdb.Subscribe(ctx, b.channel)
	if _, err := sub.Receive(ctx); err != nil {
		_ = sub.Close()
		return fmt.Errorf("cache pubsub subscribe: %w", err)
	}
	ch := sub.Channel()
	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				if m == nil {
					continue
				}
				var msg InvalidateMsg
				if err := json.Unmarshal([]byte(m.Payload), &msg); err != nil {
					continue
				}
				if msg.Origin == origin {
					continue
				}
				handler(msg)
			}
		}
	}()
	return nil
}
