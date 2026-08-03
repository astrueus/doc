package gopool

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadOrStoreDuplicateKey(t *testing.T) {
	pool := NewChannelPool(2, 10)
	pool.Start()
	defer pool.Wait()

	var n atomic.Int32
	h := func() { n.Add(1) }

	if err := pool.LoadOrStore("key1", h); err != nil {
		t.Fatalf("first LoadOrStore: %v", err)
	}
	if err := pool.LoadOrStore("key1", h); err != ErrHandlerIsExist {
		t.Fatalf("duplicate key: got %v want ErrHandlerIsExist", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for n.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n.Load() != 1 {
		t.Fatalf("handler run count: got %d want 1", n.Load())
	}
}
