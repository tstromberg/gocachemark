package cache

import (
	"sync"

	"github.com/dgryski/go-s4lru"
)

type s4lruCache struct {
	mu sync.Mutex
	c  *s4lru.Cache
}

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
	return v.(string), true
}

func (c *s4lruCache) Set(key, value string) {
	c.mu.Lock()
	c.c.Set(key, value)
	c.mu.Unlock()
}

func (c *s4lruCache) Name() string {
	return "s4lru"
}

func (c *s4lruCache) Close() {}
