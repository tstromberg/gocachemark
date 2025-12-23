package cache

import (
	"sync"

	"github.com/Code-Hex/go-generics-cache/policy/clock"
)

type clockCache struct {
	mu sync.Mutex
	c  *clock.Cache[string, string]
}

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

func (c *clockCache) Name() string {
	return "clock"
}

func (c *clockCache) Close() {}
