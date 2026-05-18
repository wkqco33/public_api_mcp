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
	mu      sync.Mutex
	entries map[K]entry[V]
}

func New[K comparable, V any]() *TTLCache[K, V] {
	return &TTLCache[K, V]{entries: make(map[K]entry[V])}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(e.expiresAt) {
		delete(c.entries, key)
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

func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}
