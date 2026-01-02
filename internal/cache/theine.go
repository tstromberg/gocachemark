package cache

import (
	"context"
	"sync"

	"github.com/Yiling-J/theine-go"
)

// theineCache uses the regular Cache for Get/Set benchmarks (hit rate, latency, throughput).
// For GetOrSet benchmarks, it uses a separate LoadingCache to test theine's native singleflight.
type theineCache struct {
	c       *theine.Cache[string, string]
	loadC   *theine.LoadingCache[string, string]
	pending sync.Map
}

// NewTheine creates a Theine cache.
func NewTheine(capacity int) Cache {
	c, _ := theine.NewBuilder[string, string](int64(capacity)).Build() //nolint:errcheck // capacity always valid
	tc := &theineCache{c: c}
	lc, _ := theine.NewBuilder[string, string](int64(capacity)).BuildWithLoader( //nolint:errcheck // capacity always valid
		func(_ context.Context, key string) (theine.Loaded[string], error) {
			if v, ok := tc.pending.LoadAndDelete(key); ok {
				return theine.Loaded[string]{Value: v.(string), Cost: 1, TTL: 0}, nil //nolint:errcheck // type known
			}
			return theine.Loaded[string]{Value: key, Cost: 1, TTL: 0}, nil
		},
	)
	tc.loadC = lc
	return tc
}

func (c *theineCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *theineCache) Set(key, value string) {
	c.c.Set(key, value, 1)
	c.loadC.Set(key, value, 1)
}

func (*theineCache) Name() string {
	return "theine"
}

func (c *theineCache) Close() {
	c.c.Close()
	c.loadC.Close()
}

func (c *theineCache) GetOrSet(key, value string) string {
	c.pending.Store(key, value)
	v, _ := c.loadC.Get(context.Background(), key) //nolint:errcheck // loader never fails
	return v
}

// theineIntCache for int-keyed benchmarks.
type theineIntCache struct {
	c       *theine.Cache[int, int]
	loadC   *theine.LoadingCache[int, int]
	pending sync.Map
}

// NewTheineInt creates a Theine cache with int keys.
func NewTheineInt(capacity int) IntCache {
	c, _ := theine.NewBuilder[int, int](int64(capacity)).Build() //nolint:errcheck // capacity always valid
	tc := &theineIntCache{c: c}
	lc, _ := theine.NewBuilder[int, int](int64(capacity)).BuildWithLoader( //nolint:errcheck // capacity always valid
		func(_ context.Context, key int) (theine.Loaded[int], error) {
			if v, ok := tc.pending.LoadAndDelete(key); ok {
				return theine.Loaded[int]{Value: v.(int), Cost: 1, TTL: 0}, nil //nolint:errcheck // type known
			}
			return theine.Loaded[int]{Value: key, Cost: 1, TTL: 0}, nil
		},
	)
	tc.loadC = lc
	return tc
}

func (c *theineIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *theineIntCache) Set(key, value int) {
	c.c.Set(key, value, 1)
	c.loadC.Set(key, value, 1)
}

func (*theineIntCache) Name() string {
	return "theine"
}

func (c *theineIntCache) Close() {
	c.c.Close()
	c.loadC.Close()
}

func (c *theineIntCache) GetOrSet(key, value int) int {
	c.pending.Store(key, value)
	v, _ := c.loadC.Get(context.Background(), key) //nolint:errcheck // loader never fails
	return v
}
