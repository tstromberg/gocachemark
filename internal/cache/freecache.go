package cache

import "github.com/coocood/freecache"

type freecacheCache struct {
	c *freecache.Cache
}

// NewFreecache creates a freecache with default entry size estimate.
// For accurate benchmarks, use NewFreecacheSized with workload-specific entry sizes.
func NewFreecache(capacity int) Cache {
	return NewFreecacheSized(capacity, 200) // conservative default
}

// NewFreecacheSized creates a freecache with a specific entry size.
// entrySize should be key + value + internal overhead (~32 bytes).
func NewFreecacheSized(capacity, entrySize int) Cache {
	cacheBytes := max(capacity*entrySize,
		// minimum 512KB
		512*1024)
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
	c.c.Set([]byte(key), []byte(value), 0) //nolint:errcheck,gosec // best-effort set
}

func (*freecacheCache) Name() string {
	return "freecache"
}

func (*freecacheCache) Close() {}

func (c *freecacheCache) GetOrSet(key, value string) string {
	result, _ := c.c.GetOrSet([]byte(key), []byte(value), 0) //nolint:errcheck // best-effort
	return string(result)
}
