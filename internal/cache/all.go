package cache

// registry maps cache names to their factory functions.
var registry = map[string]Factory{
	"multicache":    NewMulticache,
	"otter":         NewOtter,
	"theine":        NewTheine,
	"ttlcache":      NewTTLCache,
	"ristretto":     NewRistretto,
	"tinylfu":       NewTinyLFU,
	"sieve":         NewSieve,
	"s3-fifo":       NewS3FIFO,
	"freelru-shard": NewFreeLRUSharded,
	"freelru-sync":  NewFreeLRUSynced,
	"freecache":     NewFreecache,
	"2q":            NewTwoQueue,
	"s4lru":         NewS4LRU,
	"clock":         NewClock,
	"lru":           NewLRU,
}

// sizedRegistry maps cache names to their sized factory functions.
// Only caches that need entry size information are included.
var sizedRegistry = map[string]SizedFactory{
	"freecache": NewFreecacheSized,
}

// intRegistry maps cache names to their int-keyed factory functions.
var intRegistry = map[string]IntFactory{
	"multicache":    NewMulticacheInt,
	"otter":         NewOtterInt,
	"theine":        NewTheineInt,
	"ttlcache":      NewTTLCacheInt,
	"freelru-shard": NewFreeLRUShardedInt,
}

// defaultOrder defines the display order for caches.
var defaultOrder = []string{
	"multicache",
	"otter", "theine", "ttlcache", "ristretto", "tinylfu", "sieve", "s3-fifo",
	"freelru-shard", "freelru-sync", "freecache", "2q", "s4lru", "clock", "lru",
}

// intOrder defines the display order for int-keyed caches.
var intOrder = []string{
	"multicache", "otter", "theine", "ttlcache", "freelru-shard",
}

// Filter holds the current cache filter (nil = all caches).
var Filter map[string]bool

// SetFilter sets which caches to include in benchmarks.
func SetFilter(names []string) {
	if len(names) == 0 {
		Filter = nil
		return
	}
	Filter = make(map[string]bool)
	for _, name := range names {
		Filter[name] = true
	}
}

// All returns factories for all (or filtered) cache implementations.
func All() []Factory {
	var factories []Factory
	for _, name := range defaultOrder {
		if Filter != nil && !Filter[name] {
			continue
		}
		if f, ok := registry[name]; ok {
			factories = append(factories, f)
		}
	}
	return factories
}

// AllNames returns the names of all (or filtered) cache implementations.
func AllNames() []string {
	if Filter == nil {
		return defaultOrder
	}
	var names []string
	for _, name := range defaultOrder {
		if Filter[name] {
			names = append(names, name)
		}
	}
	return names
}

// AvailableNames returns all available cache names (ignoring filter).
func AvailableNames() []string {
	return defaultOrder
}

// AllInt returns factories for all (or filtered) int-keyed cache implementations.
func AllInt() []IntFactory {
	var factories []IntFactory
	for _, name := range intOrder {
		if Filter != nil && !Filter[name] {
			continue
		}
		if f, ok := intRegistry[name]; ok {
			factories = append(factories, f)
		}
	}
	return factories
}

// AllWithEntrySize returns factories for all caches, using the specified entry size
// for byte-based caches like freecache. Other caches ignore the entry size.
func AllWithEntrySize(entrySize int) []Factory {
	var factories []Factory
	for _, name := range defaultOrder {
		if Filter != nil && !Filter[name] {
			continue
		}
		// Use sized factory if available, otherwise use regular factory
		if sf, ok := sizedRegistry[name]; ok {
			// Wrap sized factory to match Factory signature
			// Capture sf in local variable to avoid closure bug
			sizedFactory := sf
			factories = append(factories, func(capacity int) Cache {
				return sizedFactory(capacity, entrySize)
			})
		} else if f, ok := registry[name]; ok {
			factories = append(factories, f)
		}
	}
	return factories
}
