package risk

import (
	"sync"
	"time"
)

// RateLimiter 是 in-memory 滑动窗口实现，零依赖；进程重启即清空。
type RateLimiter struct {
	max    int
	window time.Duration

	mu  sync.Mutex
	hit map[string][]int64
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, window: window, hit: map[string][]int64{}}
}

func (r *RateLimiter) Allow(key string) bool {
	if r == nil || r.max <= 0 || r.window <= 0 {
		return true
	}
	now := time.Now().UnixNano()
	cutoff := now - int64(r.window)

	r.mu.Lock()
	defer r.mu.Unlock()

	hits := r.hit[key]
	out := hits[:0]
	for _, ts := range hits {
		if ts > cutoff {
			out = append(out, ts)
		}
	}
	if len(out) >= r.max {
		r.hit[key] = out
		return false
	}
	r.hit[key] = append(out, now)
	return true
}
