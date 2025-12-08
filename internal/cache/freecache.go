package cache

import "github.com/coocood/freecache"

type freecacheCache struct {
	c *freecache.Cache
}

func NewFreecache(capacity int) Cache {
	// freecache uses bytes; estimate ~100 bytes per entry for string keys/values
	cacheBytes := capacity * 100
	if cacheBytes < 512*1024 {
		cacheBytes = 512 * 1024 // minimum 512KB
	}
	return &freecacheCache{c: freecache.NewCache(cacheBytes)}
}

func (c *freecacheCache) Get(key string) (string, bool) {
	v, err := c.c.Get([]byte(key))
	if err != nil {
		return "", false
	}
	return string(v), true
}

func (c *freecacheCache) Set(key, value string) {
	_ = c.c.Set([]byte(key), []byte(value), 0)
}

func (c *freecacheCache) Name() string {
	return "freecache"
}

func (c *freecacheCache) Close() {}
