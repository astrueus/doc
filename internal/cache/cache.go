package cache

import (
	"context"
	"time"

	beegocache "github.com/beego/beego/v2/client/cache"
)

// Init wires a Beego cache backend as the process-wide Default Cache.
func Init(c beegocache.Cache) {
	Default = newAdapter(c)
}

// Get loads key into dst (msgpack). Prefer passing a real request ctx when available.
func Get(ctx context.Context, key string, dst any) error {
	return Default.Get(ctx, key, dst)
}

// Put stores val under key for ttl (msgpack). Alias of Set for existing callers.
func Put(ctx context.Context, key string, val any, ttl time.Duration) error {
	return Default.Set(ctx, key, val, ttl)
}

// Set stores val under key for ttl (msgpack).
func Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	return Default.Set(ctx, key, val, ttl)
}

// Delete removes one or more keys.
func Delete(ctx context.Context, keys ...string) error {
	return Default.Delete(ctx, keys...)
}

// Incr increments an integer key.
func Incr(ctx context.Context, key string) error {
	return Default.Incr(ctx, key)
}

// Decr decrements an integer key.
func Decr(ctx context.Context, key string) error {
	return Default.Decr(ctx, key)
}

// IsExist reports whether key exists.
func IsExist(ctx context.Context, key string) (bool, error) {
	return Default.IsExist(ctx, key)
}

// ClearAll clears the whole cache store.
func ClearAll(ctx context.Context) error {
	return Default.Clear(ctx)
}
