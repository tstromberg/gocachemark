package cache

import "github.com/codeGROOVE-dev/sfcache"

type sfcacheCache struct {
	c *sfcache.MemoryCache[string, string]
}

func NewSFCache(capacity int) Cache {
	return &sfcacheCache{c: sfcache.New[string, string](sfcache.Size(capacity))}
}

func (c *sfcacheCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *sfcacheCache) Set(key, value string) {
	c.c.Set(key, value)
}

func (c *sfcacheCache) Name() string {
	return "sfcache"
}

func (c *sfcacheCache) Close() {
	c.c.Close()
}

type sfcacheIntCache struct {
	c *sfcache.MemoryCache[int, int]
}

func NewSFCacheInt(capacity int) IntCache {
	return &sfcacheIntCache{c: sfcache.New[int, int](sfcache.Size(capacity))}
}

func (c *sfcacheIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *sfcacheIntCache) Set(key, value int) {
	c.c.Set(key, value)
}

func (c *sfcacheIntCache) Name() string {
	return "sfcache"
}

func (c *sfcacheIntCache) Close() {
	c.c.Close()
}
