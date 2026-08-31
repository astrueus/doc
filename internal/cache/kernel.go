package cache

import "sync"

var (
	kernelMu sync.RWMutex
	kernel   *Runtime
)

// SetKernel 设置进程内 Aside Runtime；传入 nil 会关闭旧实例。
func SetKernel(rt *Runtime) {
	kernelMu.Lock()
	defer kernelMu.Unlock()
	if kernel != nil && kernel != rt {
		_ = kernel.Close()
	}
	kernel = rt
}

// Kernel 返回当前 Aside Runtime；未 Open 或已关闭时为 nil。
func Kernel() *Runtime {
	kernelMu.RLock()
	defer kernelMu.RUnlock()
	return kernel
}
