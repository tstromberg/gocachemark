package cache

import (
	"sync"

	"github.com/dgryski/go-s4lru"
)

type s4lruCache struct {
	c  *s4lru.Cache
	mu sync.Mutex
}

// NewS4LRU creates a segmented LRU cache.
func NewS4LRU(capacity int) Cache {
	return &s4lruCache{c: s4lru.New(capacity)}
}

func (c *s4lruCache) Get(key string) (string, bool) {
	c.mu.Lock()
	v, ok := c.c.Get(key)
	c.mu.Unlock()
	if !ok {
		return "", false
	}
	return v.(string), true //nolint:errcheck,revive // type is known from Set
}

func (c *s4lruCache) Set(key, value string) {
	c.mu.Lock()
	c.c.Set(key, value)
	c.mu.Unlock()
}

func (*s4lruCache) Name() string {
	return "s4lru"
}

func (*s4lruCache) Close() {}
