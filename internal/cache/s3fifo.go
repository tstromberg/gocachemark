package cache

import (
	"github.com/scalalang2/golang-fifo/s3fifo"
)

type s3fifoCache struct {
	c *s3fifo.S3FIFO[string, string]
}

func NewS3FIFO(capacity int) Cache {
	return &s3fifoCache{c: s3fifo.New[string, string](capacity, 0)}
}

func (c *s3fifoCache) Get(key string) (string, bool) {
	return c.c.Get(key)
}

func (c *s3fifoCache) Set(key, value string) {
	c.c.Set(key, value)
}

func (c *s3fifoCache) Name() string {
	return "s3-fifo"
}

func (c *s3fifoCache) Close() {}
