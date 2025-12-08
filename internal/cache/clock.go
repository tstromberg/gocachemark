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
	defer c.mu.Unlock()
	return c.c.Get(key)
}

func (c *clockCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.c.Set(key, value)
}

func (c *clockCache) Name() string {
	return "clock"
}

func (c *clockCache) Close() {}
