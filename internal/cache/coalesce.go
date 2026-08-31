package cache

import "golang.org/x/sync/singleflight"

// Coalesce 按 key 合并并发回源，防止击穿。
type Coalesce struct {
	g singleflight.Group
}

// Do 同一 key 同时只执行一次 fn；其余等待共享结果。
func (c *Coalesce) Do(key string, fn func() (any, error)) (v any, err error, shared bool) {
	return c.g.Do(key, fn)
}
