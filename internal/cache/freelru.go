package cache

import (
	lru "github.com/elastic/go-freelru"
	"github.com/zeebo/xxh3"
)

func hash(s string) uint32 {
	return uint32(xxh3.HashString(s)) //nolint:gosec // safe truncation
}

func hashInt(i int) uint32 {
	return uint32(i) //nolint:gosec // safe truncation for cache keys
}

type freeLRUSyncedCache struct {
	c *lru.SyncedLRU[string, string]
}

// NewFreeLRUSynced creates a synchronized FreeLRU cache.
func NewFreeLRUSynced(capacity int) Cache {
	c, _ := lru.NewSynced[string, string](uint32(capacity), hash) //nolint:errcheck,gosec // capacity always valid
	return &freeLRUSyncedCache{c: c}
}

func (c *freeLRUSyncedCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *freeLRUSyncedCache) Set(key, value string) {
	c.c.Add(key, value)
}

func (*freeLRUSyncedCache) Name() string {
	return "freelru-sync"
}

func (*freeLRUSyncedCache) Close() {}

type freeLRUShardedCache struct {
	c *lru.ShardedLRU[string, string]
}

// NewFreeLRUSharded creates a sharded FreeLRU cache.
func NewFreeLRUSharded(capacity int) Cache {
	c, _ := lru.NewSharded[string, string](uint32(capacity), hash) //nolint:errcheck,gosec // capacity always valid
	return &freeLRUShardedCache{c: c}
}

func (c *freeLRUShardedCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *freeLRUShardedCache) Set(key, value string) {
	c.c.Add(key, value)
}

func (*freeLRUShardedCache) Name() string {
	return "freelru-shard"
}

func (*freeLRUShardedCache) Close() {}

type freeLRUShardedIntCache struct {
	c *lru.ShardedLRU[int, int]
}

// NewFreeLRUShardedInt creates a sharded FreeLRU cache with int keys.
func NewFreeLRUShardedInt(capacity int) IntCache {
	c, _ := lru.NewSharded[int, int](uint32(capacity), hashInt) //nolint:errcheck,gosec // capacity always valid
	return &freeLRUShardedIntCache{c: c}
}

func (c *freeLRUShardedIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *freeLRUShardedIntCache) Set(key, value int) {
	c.c.Add(key, value)
}

func (*freeLRUShardedIntCache) Name() string {
	return "freelru-shard"
}

func (*freeLRUShardedIntCache) Close() {}
