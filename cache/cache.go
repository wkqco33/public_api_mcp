package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

type TTLCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]entry[V]
}

func New[K comparable, V any]() *TTLCache[K, V] {
	return &TTLCache[K, V]{entries: make(map[K]entry[V])}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		var zero V
		return zero, false
	}
	return e.value, true
}

func (c *TTLCache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry[V]{value: value, expiresAt: time.Now().Add(ttl)}
}
