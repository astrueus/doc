package store

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 为 L2 Store。key 原样写入（由 Aside KeyBuilder 带前缀）。
type Redis struct {
	rdb *redis.Client
}

// NewRedis 包装已有 go-redis 客户端。
func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{rdb: rdb}
}

func (s *Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s == nil || s.rdb == nil {
		return nil, false, nil
	}
	b, err := s.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

func (s *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	if ttl < 0 {
		ttl = 0
	}
	return s.rdb.Set(ctx, key, value, ttl).Err()
}

func (s *Redis) Del(ctx context.Context, keys ...string) error {
	if s == nil || s.rdb == nil || len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}
