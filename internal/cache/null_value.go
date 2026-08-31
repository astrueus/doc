package cache

import "errors"

// ErrNotFound 表示 loader 确认资源不存在，可按 Options.CacheNull 写入负缓存。
var ErrNotFound = errors.New("cache: not found")
