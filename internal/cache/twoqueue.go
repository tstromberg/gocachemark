package cache

import lru "github.com/hashicorp/golang-lru/v2"

type twoQueueCache struct {
	c *lru.TwoQueueCache[string, string]
}

func NewTwoQueue(capacity int) Cache {
	c, _ := lru.New2Q[string, string](capacity)
	return &twoQueueCache{c: c}
}

func (c *twoQueueCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *twoQueueCache) Set(key, value string) {
	c.c.Add(key, value)
}

func (c *twoQueueCache) Name() string {
	return "2q"
}

func (c *twoQueueCache) Close() {}
