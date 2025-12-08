package cache

import (
	lru "github.com/elastic/go-freelru"
	"github.com/zeebo/xxh3"
)

func hash(s string) uint32 {
	return uint32(xxh3.HashString(s))
}

func hashInt(i int) uint32 {
	return uint32(i)
}

type freeLRUSyncedCache struct {
	c *lru.SyncedLRU[string, string]
}

func NewFreeLRUSynced(capacity int) Cache {
	c, _ := lru.NewSynced[string, string](uint32(capacity), hash)
	return &freeLRUSyncedCache{c: c}
}

func (c *freeLRUSyncedCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *freeLRUSyncedCache) Set(key, value string) {
	c.c.Add(key, value)
}

func (c *freeLRUSyncedCache) Name() string {
	return "freelru-sync"
}

func (c *freeLRUSyncedCache) Close() {}

type freeLRUShardedCache struct {
	c *lru.ShardedLRU[string, string]
}

func NewFreeLRUSharded(capacity int) Cache {
	c, _ := lru.NewSharded[string, string](uint32(capacity), hash)
	return &freeLRUShardedCache{c: c}
}

func (c *freeLRUShardedCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *freeLRUShardedCache) Set(key, value string) {
	c.c.Add(key, value)
}

func (c *freeLRUShardedCache) Name() string {
	return "freelru-shard"
}

func (c *freeLRUShardedCache) Close() {}

type freeLRUShardedIntCache struct {
	c *lru.ShardedLRU[int, int]
}

func NewFreeLRUShardedInt(capacity int) IntCache {
	c, _ := lru.NewSharded[int, int](uint32(capacity), hashInt)
	return &freeLRUShardedIntCache{c: c}
}

func (c *freeLRUShardedIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *freeLRUShardedIntCache) Set(key, value int) {
	c.c.Add(key, value)
}

func (c *freeLRUShardedIntCache) Name() string {
	return "freelru-shard"
}

func (c *freeLRUShardedIntCache) Close() {}
