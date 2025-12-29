package cache

import (
	"context"
	"time"

	"github.com/hyp3rd/hypercache"
	"github.com/hyp3rd/hypercache/pkg/backend"
)

type hypercacheCache struct {
	c *hypercache.HyperCache[backend.InMemory]
}

// NewHypercache creates a HyperCache instance.
func NewHypercache(capacity int) Cache {
	config := hypercache.NewConfig[backend.InMemory]("in-memory")
	config.InMemoryOptions = []backend.Option[backend.InMemory]{
		backend.WithCapacity[backend.InMemory](capacity),
	}
	config.HyperCacheOptions = []hypercache.Option[backend.InMemory]{
		hypercache.WithEvictionInterval[backend.InMemory](0), // Enable proactive eviction
	}
	c, err := hypercache.New(context.Background(), hypercache.GetDefaultManager(), config)
	if err != nil {
		panic(err)
	}
	return &hypercacheCache{c: c}
}

func (c *hypercacheCache) Get(key string) (string, bool) {
	v, ok := c.c.Get(context.Background(), key)
	if !ok {
		return "", false
	}
	return v.(string), true //nolint:errcheck,revive // type is known from Set
}

func (c *hypercacheCache) Set(key, value string) {
	c.c.Set(context.Background(), key, value, 0) //nolint:errcheck,gosec // best-effort set
}

func (*hypercacheCache) Name() string {
	return "hypercache"
}

func (c *hypercacheCache) Close() {
	c.c.Stop(context.Background()) //nolint:errcheck,gosec // best-effort stop
}

func (c *hypercacheCache) GetOrSet(key, value string) string {
	result, err := c.c.GetOrSet(context.Background(), key, value, time.Hour)
	if err != nil {
		return value
	}
	return result.(string) //nolint:errcheck,revive // type is known from Set
}
