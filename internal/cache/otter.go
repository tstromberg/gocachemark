package cache

import (
	"context"

	"github.com/maypok86/otter/v2"
)

type stringLoader struct {
	value string
}

func (l *stringLoader) Load(_ context.Context, _ string) (string, error) {
	return l.value, nil
}

func (l *stringLoader) Reload(_ context.Context, _ string, _ string) (string, error) {
	return l.value, nil
}

type intLoader struct {
	value int
}

func (l *intLoader) Load(_ context.Context, _ int) (int, error) {
	return l.value, nil
}

func (l *intLoader) Reload(_ context.Context, _ int, _ int) (int, error) {
	return l.value, nil
}

type otterCache struct {
	c *otter.Cache[string, string]
}

// NewOtter creates an Otter cache.
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

func (*otterCache) Name() string {
	return "otter"
}

func (*otterCache) Close() {}

func (c *otterCache) GetOrSet(key, value string) string {
	result, _ := c.c.Get(context.Background(), key, &stringLoader{value: value}) //nolint:errcheck // loader never fails
	return result
}

type otterIntCache struct {
	c *otter.Cache[int, int]
}

// NewOtterInt creates an Otter cache with int keys.
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

func (*otterIntCache) Name() string {
	return "otter"
}

func (*otterIntCache) Close() {}

func (c *otterIntCache) GetOrSet(key, value int) int {
	result, _ := c.c.Get(context.Background(), key, &intLoader{value: value}) //nolint:errcheck // loader never fails
	return result
}
