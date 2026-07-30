package mcp

import (
	"sync"
	"time"

	"git.itopcms.com/jackliu/doc/internal/config"
	"golang.org/x/time/rate"
)

type tokenLimits struct {
	read   *rate.Limiter
	write  *rate.Limiter
	delete *rate.Limiter
}

type tokenRateLimiter struct {
	mu    sync.Mutex
	byID  map[int]*tokenLimits
	read  int
	write int
	del   int
}

var globalRateLimiter = newTokenRateLimiter(60)

func newTokenRateLimiter(perMin int) *tokenRateLimiter {
	if perMin <= 0 {
		perMin = 60
	}
	write := perMin / 2
	if write < 1 {
		write = 1
	}
	del := perMin / 6
	if del < 1 {
		del = 1
	}
	return &tokenRateLimiter{
		byID:  make(map[int]*tokenLimits),
		read:  perMin,
		write: write,
		del:   del,
	}
}

func configureRateLimiterFromConfig() {
	n := 60
	if config.Global != nil && config.Global.MCP.RateLimit > 0 {
		n = config.Global.MCP.RateLimit
	}
	globalRateLimiter = newTokenRateLimiter(n)
}

func (r *tokenRateLimiter) getOrCreate(tokenID int) *tokenLimits {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.byID[tokenID]; ok {
		return l
	}
	l := &tokenLimits{
		read:   rate.NewLimiter(perMinute(r.read), r.read),
		write:  rate.NewLimiter(perMinute(r.write), r.write),
		delete: rate.NewLimiter(perMinute(r.del), r.del),
	}
	r.byID[tokenID] = l
	return l
}

func perMinute(n int) rate.Limit {
	if n <= 0 {
		n = 1
	}
	return rate.Every(time.Minute / time.Duration(n))
}

// AllowByToken enforces per-token limits: read=X/min, write=X/2, delete=X/6.
func AllowByToken(tokenID int, toolKind string) bool {
	if tokenID <= 0 {
		// Anonymous / stdio-member fallback: use a shared bucket keyed by 0.
		tokenID = 0
	}
	l := globalRateLimiter.getOrCreate(tokenID)
	switch toolKind {
	case "delete":
		return l.delete.Allow()
	case "write":
		return l.write.Allow()
	default:
		return l.read.Allow()
	}
}
