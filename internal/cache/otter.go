package cache

import "github.com/maypok86/otter/v2"

type otterCache struct {
	c *otter.Cache[string, string]
}

func NewOtter(capacity int) Cache {
	c := otter.Must(&otter.Options[string, string]{MaximumSize: capacity})
	return &otterCache{c: c}
}

func (c *otterCache) Get(key string) (string, bool) {
	return c.c.GetIfPresent(key)
}

func (c *otterCache) Set(key, value string) {
	c.c.Set(key, value)
}

func (c *otterCache) Name() string {
	return "otter"
}

func (c *otterCache) Close() {}

type otterIntCache struct {
	c *otter.Cache[int, int]
}

func NewOtterInt(capacity int) IntCache {
	c := otter.Must(&otter.Options[int, int]{MaximumSize: capacity})
	return &otterIntCache{c: c}
}

func (c *otterIntCache) Get(key int) (int, bool) {
	return c.c.GetIfPresent(key)
}

func (c *otterIntCache) Set(key, value int) {
	c.c.Set(key, value)
}

func (c *otterIntCache) Name() string {
	return "otter"
}

func (c *otterIntCache) Close() {}
