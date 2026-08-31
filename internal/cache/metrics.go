package cache

import "sync/atomic"

// Metrics 为进程内计数；用 Snapshot().Map() 写入日志字段。
type Metrics struct {
	L1Hit       atomic.Int64
	L2Hit       atomic.Int64
	Miss        atomic.Int64
	Load        atomic.Int64
	LoadShared  atomic.Int64
	LoadErr     atomic.Int64
	NullHit     atomic.Int64
	SoftRefresh atomic.Int64
}

// Snapshot 返回当前计数副本。
func (m *Metrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	return MetricsSnapshot{
		L1Hit:       m.L1Hit.Load(),
		L2Hit:       m.L2Hit.Load(),
		Miss:        m.Miss.Load(),
		Load:        m.Load.Load(),
		LoadShared:  m.LoadShared.Load(),
		LoadErr:     m.LoadErr.Load(),
		NullHit:     m.NullHit.Load(),
		SoftRefresh: m.SoftRefresh.Load(),
	}
}

// MetricsSnapshot 为 Metrics 的只读拷贝。
type MetricsSnapshot struct {
	L1Hit       int64
	L2Hit       int64
	Miss        int64
	Load        int64
	LoadShared  int64
	LoadErr     int64
	NullHit     int64
	SoftRefresh int64
}

// Map 返回可写入日志的稳定字段名。
func (s MetricsSnapshot) Map() map[string]int64 {
	return map[string]int64{
		"cache_l1_hit":       s.L1Hit,
		"cache_l2_hit":       s.L2Hit,
		"cache_miss":         s.Miss,
		"cache_load":         s.Load,
		"cache_load_shared":  s.LoadShared,
		"cache_load_err":     s.LoadErr,
		"cache_null_hit":     s.NullHit,
		"cache_soft_refresh": s.SoftRefresh,
	}
}
