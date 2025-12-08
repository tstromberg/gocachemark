package cache

import "github.com/coocood/freecache"

// freecache internal overhead per entry (header, alignment, etc.)
const freecacheOverhead = 32

type freecacheCache struct {
	c *freecache.Cache
}

// NewFreecache creates a freecache with default entry size estimate.
// For accurate benchmarks, use NewFreecacheSized with workload-specific entry sizes.
func NewFreecache(capacity int) Cache {
	return NewFreecacheSized(capacity, 200) // conservative default
}

// NewFreecacheSized creates a freecache with a specific entry size.
// entrySize should be key + value + freecacheOverhead.
func NewFreecacheSized(capacity, entrySize int) Cache {
	cacheBytes := capacity * entrySize
	if cacheBytes < 512*1024 {
		cacheBytes = 512 * 1024 // minimum 512KB
	}
	return &freecacheCache{c: freecache.NewCache(cacheBytes)}
}

// FreecacheSizedFactory returns a SizedFactory for freecache.
func FreecacheSizedFactory() SizedFactory {
	return NewFreecacheSized
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

func (c *freecacheCache) GetOrSet(key, value string) string {
	result, _ := c.c.GetOrSet([]byte(key), []byte(value), 0)
	return string(result)
}
