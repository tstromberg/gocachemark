package cache

import "github.com/Yiling-J/theine-go"

type theineCache struct {
	c *theine.Cache[string, string]
}

func NewTheine(capacity int) Cache {
	c, _ := theine.NewBuilder[string, string](int64(capacity)).Build()
	return &theineCache{c: c}
}

func (c *theineCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *theineCache) Set(key, value string) {
	c.c.Set(key, value, 1)
}

func (c *theineCache) Name() string {
	return "theine"
}

func (c *theineCache) Close() {
	c.c.Close()
}

type theineIntCache struct {
	c *theine.Cache[int, int]
}

func NewTheineInt(capacity int) IntCache {
	c, _ := theine.NewBuilder[int, int](int64(capacity)).Build()
	return &theineIntCache{c: c}
}

func (c *theineIntCache) Get(key int) (int, bool) {
	return c.c.Get(key)
}

func (c *theineIntCache) Set(key, value int) {
	c.c.Set(key, value, 1)
}

func (c *theineIntCache) Name() string {
	return "theine"
}

func (c *theineIntCache) Close() {
	c.c.Close()
}
