package cache

import (
	"context"
	"time"
)

// Cache is the application-facing cache API.
// Values are encoded with msgpack (not gob), so entries do not bind to Go package paths.
type Cache interface {
	Get(ctx context.Context, key string, dst any) error
	Set(ctx context.Context, key string, val any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	IsExist(ctx context.Context, key string) (bool, error)
	Incr(ctx context.Context, key string) error
	Decr(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// Default 为进程级 beego 适配实例（T12-d 起业务不再读写）。
var Default Cache
