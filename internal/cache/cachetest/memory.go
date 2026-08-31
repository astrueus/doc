package cachetest

import "git.itopcms.com/astrueus/doc/internal/cache/store"

// MemoryPair 返回一对内存 Store，用作 Aside 的 L1 / L2。
func MemoryPair() (l1, l2 store.Store) {
	return store.NewMemory(), store.NewMemory()
}
