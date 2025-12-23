package cache

import "github.com/dgraph-io/ristretto"

type ristrettoCache struct {
	c *ristretto.Cache
}

func NewRistretto(capacity int) Cache {
	c, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters:        int64(capacity) * 10,
		MaxCost:            int64(capacity),
		BufferItems:        64,
		IgnoreInternalCost: true,
	})
	return &ristrettoCache{c: c}
}

func (c *ristrettoCache) Get(key string) (string, bool) {
	v, ok := c.c.Get(key)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (c *ristrettoCache) Set(key, value string) {
	c.c.Set(key, value, 1)
}

func (c *ristrettoCache) Name() string {
	return "ristretto"
}

func (c *ristrettoCache) Close() {
	c.c.Wait() // flush pending async writes
	c.c.Close()
}
