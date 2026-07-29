package cache

import (
	"context"
	"errors"
	"time"

	beegocache "github.com/beego/beego/v2/client/cache"
	"github.com/beego/beego/v2/core/logs"
	"github.com/vmihailenco/msgpack/v5"
)

// ErrNotExist indicates the key is missing or holds an empty value.
var ErrNotExist = errors.New("cache does not exist")

// adapter wraps a Beego cache backend and serializes values with msgpack.
type adapter struct {
	bm beegocache.Cache
}

func newAdapter(bm beegocache.Cache) *adapter {
	return &adapter{bm: bm}
}

func (a *adapter) Get(ctx context.Context, key string, dst any) error {
	val, err := a.bm.Get(ctx, key)
	if err != nil {
		return errors.New("get cache error:" + err.Error())
	}
	if val == nil {
		return ErrNotExist
	}

	var raw []byte
	switch v := val.(type) {
	case []byte:
		raw = v
	case string:
		if v == "" {
			return ErrNotExist
		}
		raw = []byte(v)
	default:
		return errors.New("value is not []byte or string")
	}

	if err := msgpack.Unmarshal(raw, dst); err != nil {
		logs.Error("反序列化对象失败 ->", err)
		return err
	}
	return nil
}

func (a *adapter) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	raw, err := msgpack.Marshal(val)
	if err != nil {
		logs.Error("序列化对象失败 ->", err)
		return err
	}
	return a.bm.Put(ctx, key, string(raw), ttl)
}

func (a *adapter) Delete(ctx context.Context, keys ...string) error {
	var first error
	for _, key := range keys {
		if err := a.bm.Delete(ctx, key); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (a *adapter) IsExist(ctx context.Context, key string) (bool, error) {
	return a.bm.IsExist(ctx, key)
}

func (a *adapter) Incr(ctx context.Context, key string) error {
	return a.bm.Incr(ctx, key)
}

func (a *adapter) Decr(ctx context.Context, key string) error {
	return a.bm.Decr(ctx, key)
}

func (a *adapter) Clear(ctx context.Context) error {
	return a.bm.ClearAll(ctx)
}
