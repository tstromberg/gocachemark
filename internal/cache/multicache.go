package cache

import "github.com/codeGROOVE-dev/multicache"

type multicacheCache struct {
	c *multicache.Cache[string, string]
}

func NewMulticache(capacity int) Cache {
	return &multicacheCache{c: multicache.New[string, string](multicache.Size(capacity))}
}

func (c *multicacheCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *multicacheCache) Set(key, value string) {
	c.c.Set(key, value)
}

func (c *multicacheCache) Name() string {
	return "multicache"
}

func (c *multicacheCache) Close() {
	c.c.Close()
}

func (c *multicacheCache) GetOrSet(key, value string) string {
	result, _ := c.c.SetIfAbsent(key, value)
	return result
}

type multicacheIntCache struct {
	c *multicache.Cache[int, int]
}

func NewMulticacheInt(capacity int) IntCache {
	return &multicacheIntCache{c: multicache.New[int, int](multicache.Size(capacity))}
}

func (c *multicacheIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *multicacheIntCache) Set(key, value int) {
	c.c.Set(key, value)
}

func (c *multicacheIntCache) Name() string {
	return "multicache"
}

func (c *multicacheIntCache) Close() {
	c.c.Close()
}

func (c *multicacheIntCache) GetOrSet(key, value int) int {
	result, _ := c.c.SetIfAbsent(key, value)
	return result
}
