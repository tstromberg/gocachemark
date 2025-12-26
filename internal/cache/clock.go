package cache

import (
	"sync"

	"github.com/Code-Hex/go-generics-cache/policy/clock"
)

type clockCache struct {
	c  *clock.Cache[string, string]
	mu sync.Mutex
}

// NewClock creates a clock-based cache.
func NewClock(capacity int) Cache {
	return &clockCache{
		c: clock.NewCache[string, string](clock.WithCapacity(capacity)),
	}
}

func (c *clockCache) Get(key string) (string, bool) {
	c.mu.Lock()
	v, ok := c.c.Get(key)
	c.mu.Unlock()
	return v, ok
}

func (c *clockCache) Set(key, value string) {
	c.mu.Lock()
	c.c.Set(key, value)
	c.mu.Unlock()
}

func (*clockCache) Name() string {
	return "clock"
}

func (*clockCache) Close() {}
