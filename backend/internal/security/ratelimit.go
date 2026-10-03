package security

import (
	"sync"
	"time"
)

type visitor struct {
	tokens     int
	lastSeen   time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // tokens per interval
	interval time.Duration
}

func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		interval: interval,
	}

	// Periodically prune stale visitors
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			limiter.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for ip, v := range limiter.visitors {
				if v.lastSeen.Before(cutoff) {
					delete(limiter.visitors, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return limiter
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{tokens: rl.rate - 1, lastSeen: now}
		return true
	}

	// Refill tokens based on elapsed intervals
	elapsed := now.Sub(v.lastSeen)
	if elapsed >= rl.interval {
		v.tokens = rl.rate
		v.lastSeen = now
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}
