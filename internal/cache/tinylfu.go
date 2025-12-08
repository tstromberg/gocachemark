package cache

import "github.com/vmihailenco/go-tinylfu"

type tinyLFUCache struct {
	c *tinylfu.SyncT
}

func NewTinyLFU(capacity int) Cache {
	return &tinyLFUCache{c: tinylfu.NewSync(capacity, capacity*10)}
}

func (c *tinyLFUCache) Get(key string) (string, bool) {
	v, ok := c.c.Get(key)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (c *tinyLFUCache) Set(key, value string) {
	c.c.Set(&tinylfu.Item{Key: key, Value: value})
}

func (c *tinyLFUCache) Name() string {
	return "tinylfu"
}

func (c *tinyLFUCache) Close() {}
