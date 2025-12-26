package cache

import lru "github.com/hashicorp/golang-lru/v2"

type lruCache struct {
	c *lru.Cache[string, string]
}

// NewLRU creates a hashicorp LRU cache.
func NewLRU(capacity int) Cache {
	c, _ := lru.New[string, string](capacity) //nolint:errcheck // capacity always valid
	return &lruCache{c: c}
}

func (c *lruCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *lruCache) Set(key, value string) {
	c.c.Add(key, value)
}

func (*lruCache) Name() string {
	return "lru"
}

func (*lruCache) Close() {}
