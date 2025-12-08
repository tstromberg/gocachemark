// Package cache provides a unified interface for benchmarking cache implementations.
package cache

// Cache is a minimal interface for cache benchmarking with string keys.
type Cache interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Name() string
	Close()
}

// Factory creates a new cache instance with the given capacity.
type Factory func(capacity int) Cache

// SizedFactory creates a new cache instance with capacity and expected entry size.
// Used for byte-based caches like freecache that need to know entry sizes.
type SizedFactory func(capacity, entrySize int) Cache

// IntCache is a minimal interface for cache benchmarking with int keys.
type IntCache interface {
	Get(key int) (int, bool)
	Set(key, value int)
	Name() string
	Close()
}

// IntFactory creates a new int-keyed cache instance with the given capacity.
type IntFactory func(capacity int) IntCache
