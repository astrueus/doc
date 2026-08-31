package store

import (
	"context"
	"time"
)

// Store 是缓存内核的字节存储。L1 / L2 都实现同一接口。
type Store interface {
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}
