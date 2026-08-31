package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisTags struct {
	rdb    *redis.Client
	prefix string
}

// NewRedisTags 用 Redis SET 维护 tag→keys。
func NewRedisTags(rdb *redis.Client, prefix string) TagIndex {
	if prefix == "" {
		prefix = DefaultKeyPrefix
	}
	return &redisTags{rdb: rdb, prefix: prefix}
}

func (t *redisTags) tagKey(tag string) string {
	return t.prefix + "tag:" + tag
}

func (t *redisTags) Add(ctx context.Context, key string, tags []string, ttl time.Duration) error {
	if t == nil || t.rdb == nil || key == "" || len(tags) == 0 {
		return nil
	}
	pipe := t.rdb.Pipeline()
	n := 0
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		tk := t.tagKey(tag)
		pipe.SAdd(ctx, tk, key)
		if ttl > 0 {
			pipe.Expire(ctx, tk, ttl)
		}
		n++
	}
	if n == 0 {
		return nil
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (t *redisTags) KeysOf(ctx context.Context, tags []string) ([]string, error) {
	if t == nil || t.rdb == nil || len(tags) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		members, err := t.rdb.SMembers(ctx, t.tagKey(tag)).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range members {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	return out, nil
}

func (t *redisTags) RemoveKeys(context.Context, []string) error {
	// 反向索引成本高；失效走 KeysOf + RemoveTags，残留 member 再 Invalidate 时 Del 幂等。
	return nil
}

func (t *redisTags) RemoveTags(ctx context.Context, tags []string) error {
	if t == nil || t.rdb == nil || len(tags) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag != "" {
			keys = append(keys, t.tagKey(tag))
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return t.rdb.Del(ctx, keys...).Err()
}
