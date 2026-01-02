package cache

import "github.com/codeGROOVE-dev/fido"

type fidoCache struct {
	c *fido.Cache[string, string]
}

// NewFido creates a Fido cache.
func NewFido(capacity int) Cache {
	return &fidoCache{c: fido.New[string, string](fido.Size(capacity))}
}

func (c *fidoCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *fidoCache) Set(key, value string) {
	c.c.Set(key, value)
}

func (*fidoCache) Name() string {
	return "fido"
}

func (*fidoCache) Close() {}

func (c *fidoCache) GetOrSet(key, value string) string {
	v, _ := c.c.Fetch(key, func() (string, error) { return value, nil }) //nolint:errcheck // loader never returns error
	return v
}

type fidoIntCache struct {
	c *fido.Cache[int, int]
}

// NewFidoInt creates a Fido cache with int keys.
func NewFidoInt(capacity int) IntCache {
	return &fidoIntCache{c: fido.New[int, int](fido.Size(capacity))}
}

func (c *fidoIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *fidoIntCache) Set(key, value int) {
	c.c.Set(key, value)
}

func (*fidoIntCache) Name() string {
	return "fido"
}

func (*fidoIntCache) Close() {}

func (c *fidoIntCache) GetOrSet(key, value int) int {
	v, _ := c.c.Fetch(key, func() (int, error) { return value, nil }) //nolint:errcheck // loader never returns error
	return v
}
