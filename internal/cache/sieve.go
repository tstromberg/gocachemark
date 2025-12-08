package cache

import (
	"github.com/scalalang2/golang-fifo/sieve"
)

type sieveCache struct {
	c *sieve.Sieve[string, string]
}

func NewSieve(capacity int) Cache {
	return &sieveCache{c: sieve.New[string, string](capacity, 0)}
}

func (c *sieveCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *sieveCache) Set(key, value string) {
	c.c.Set(key, value)
}

func (c *sieveCache) Name() string {
	return "sieve"
}

func (c *sieveCache) Close() {}
