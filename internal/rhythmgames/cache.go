package rhythmgames

import (
	"sync"
	"time"
)

type entry[T any] struct {
	val   T
	expAt time.Time
}

type TTLCache[T any] struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[string]entry[T]
}

func NewTTLCache[T any](ttl time.Duration) *TTLCache[T] {
	return &TTLCache[T]{ttl: ttl, m: make(map[string]entry[T])}
}

func (c *TTLCache[T]) Get(k string) (T, bool) {
	var zero T
	c.mu.RLock()
	e, ok := c.m[k]
	c.mu.RUnlock()
	if !ok {
		return zero, false
	}
	if c.ttl > 0 && !e.expAt.IsZero() && time.Now().After(e.expAt) {
		return zero, false
	}
	return e.val, true
}

func (c *TTLCache[T]) Set(k string, v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := time.Time{}
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}
	c.m[k] = entry[T]{val: v, expAt: exp}
}
