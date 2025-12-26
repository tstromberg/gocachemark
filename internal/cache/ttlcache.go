package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type ttlcacheCache struct {
	c *ttlcache.Cache[string, string]
}

// NewTTLCache creates a TTL-based cache.
func NewTTLCache(capacity int) Cache {
	c := ttlcache.New[string, string](
		ttlcache.WithCapacity[string, string](uint64(capacity)), //nolint:gosec // capacity always positive
		ttlcache.WithTTL[string, string](time.Hour),             // Long TTL since we're testing capacity, not expiration
	)
	go c.Start()
	return &ttlcacheCache{c: c}
}

func (c *ttlcacheCache) Get(key string) (string, bool) {
	item := c.c.Get(key)
	if item == nil {
		return "", false
	}
	return item.Value(), true
}

func (c *ttlcacheCache) Set(key, value string) {
	c.c.Set(key, value, ttlcache.DefaultTTL)
}

func (*ttlcacheCache) Name() string {
	return "ttlcache"
}

func (c *ttlcacheCache) Close() {
	c.c.Stop()
}

func (c *ttlcacheCache) GetOrSet(key, value string) string {
	item, _ := c.c.GetOrSet(key, value)
	return item.Value()
}

type ttlcacheIntCache struct {
	c *ttlcache.Cache[int, int]
}

// NewTTLCacheInt creates a TTL-based cache with int keys.
func NewTTLCacheInt(capacity int) IntCache {
	c := ttlcache.New[int, int](
		ttlcache.WithCapacity[int, int](uint64(capacity)), //nolint:gosec // capacity always positive
		ttlcache.WithTTL[int, int](time.Hour),
	)
	go c.Start()
	return &ttlcacheIntCache{c: c}
}

func (c *ttlcacheIntCache) Get(key int) (int, bool) {
	item := c.c.Get(key)
	if item == nil {
		return 0, false
	}
	return item.Value(), true
}

func (c *ttlcacheIntCache) Set(key, value int) {
	c.c.Set(key, value, ttlcache.DefaultTTL)
}

func (*ttlcacheIntCache) Name() string {
	return "ttlcache"
}

func (c *ttlcacheIntCache) Close() {
	c.c.Stop()
}

func (c *ttlcacheIntCache) GetOrSet(key, value int) int {
	item, _ := c.c.GetOrSet(key, value)
	return item.Value()
}
