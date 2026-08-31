package cache

import "sync/atomic"

// Metrics 为进程内计数，T12-b 再接到可刮取后端。
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
