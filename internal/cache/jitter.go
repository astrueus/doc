package cache

import (
	"math/rand/v2"
	"time"
)

func jitteredTTL(ttl time.Duration, jitter float64) time.Duration {
	if ttl <= 0 || jitter <= 0 {
		return ttl
	}
	f := 1 + (rand.Float64()*2-1)*jitter
	if f < 0.1 {
		f = 0.1
	}
	out := time.Duration(float64(ttl) * f)
	if out <= 0 {
		return ttl
	}
	return out
}
