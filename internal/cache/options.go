package cache

import "time"

const (
	defaultL1TTL   = 20 * time.Second
	defaultNullTTL = 45 * time.Second
)

// Options 控制单次 GetOrLoad / Set 的过期与防护策略。
type Options struct {
	// TTL 为 hard 过期；超过后必须回源。
	TTL time.Duration
	// SoftTTL 小于 TTL 时：超过后仍返回旧值，并后台重建。0 表示关闭。
	SoftTTL time.Duration
	// Jitter 为 TTL 抖动比例（如 0.1 表示 ±10%），减轻雪崩。
	Jitter float64
	// Tags 写入时登记，供 InvalidateTag 使用（T12-a 为进程内索引）。
	Tags []string
	// CacheNull 为 true 时，loader 返回 ErrNotFound 会写入负缓存。
	CacheNull bool
	// NullTTL 负缓存 TTL；0 则用 45s。
	NullTTL time.Duration
	// L1TTL 本地层 TTL；0 则用 20s，且不超过 hard TTL。
	L1TTL time.Duration
}

func (o Options) normalized() Options {
	if o.L1TTL <= 0 {
		o.L1TTL = defaultL1TTL
	}
	if o.NullTTL <= 0 {
		o.NullTTL = defaultNullTTL
	}
	if o.SoftTTL <= 0 || o.TTL <= 0 || o.SoftTTL >= o.TTL {
		o.SoftTTL = 0
	}
	if o.Jitter < 0 {
		o.Jitter = 0
	}
	return o
}

func (o Options) l1TTLCapped(hard time.Duration) time.Duration {
	if hard <= 0 {
		return o.L1TTL
	}
	if o.L1TTL < hard {
		return o.L1TTL
	}
	return hard
}
