package cache

import (
	"sync"
	"time"
)

type entry struct {
	data      string
	expiresAt time.Time
}

type FragmentCache struct {
	mu    sync.RWMutex
	items map[string]*entry
	ttl   time.Duration
}

func NewFragmentCache(ttl time.Duration) *FragmentCache {
	return &FragmentCache{
		items: make(map[string]*entry),
		ttl:   ttl,
	}
}

func (c *FragmentCache) Get(key string) (string, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return "", false
	}
	return e.data, true
}

func (c *FragmentCache) Set(key string, data string) {
	c.mu.Lock()
	c.items[key] = &entry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *FragmentCache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *FragmentCache) Cleanup() {
	now := time.Now()
	c.mu.Lock()
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}
