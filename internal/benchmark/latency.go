package benchmark

import (
	"strconv"
	"testing"

	"github.com/tstromberg/gocachemark/internal/cache"
)

// LatencyResult holds single-threaded latency results for a cache.
type LatencyResult struct {
	Name            string
	GetNsOp         float64 // nanoseconds per Get operation
	SetNsOp         float64 // nanoseconds per Set operation (no eviction)
	SetEvictNsOp    float64 // nanoseconds per Set with eviction (20x keyspace)
	GetOrSetNsOp    float64 // nanoseconds per GetOrSet operation
	GetAllocs       int64   // allocations per Get
	SetAllocs       int64   // allocations per Set
	SetEvictAllocs  int64   // allocations per Set with eviction
	GetOrSetAllocs  int64   // allocations per GetOrSet
	HasGetOrSet     bool    // whether cache supports GetOrSet
}

// IntLatencyResult holds single-threaded latency results for int-keyed caches.
type IntLatencyResult struct {
	Name           string
	GetNsOp        float64
	SetNsOp        float64
	SetEvictNsOp   float64
	GetOrSetNsOp   float64
	GetAllocs      int64
	SetAllocs      int64
	SetEvictAllocs int64
	GetOrSetAllocs int64
	HasGetOrSet    bool
}

const latencyCacheSize = 10000

// RunLatency benchmarks single-threaded Get/Set latency for all caches (string keys).
func RunLatency() []LatencyResult {
	results := make([]LatencyResult, 0, len(cache.All()))

	// Pre-generate keys once for all benchmarks
	keys := make([]string, latencyCacheSize)
	for i := range latencyCacheSize {
		keys[i] = strconv.Itoa(i)
	}
	evictKeys := make([]string, latencyCacheSize*20)
	for i := range len(evictKeys) {
		evictKeys[i] = strconv.Itoa(i)
	}

	for _, factory := range cache.All() {
		c := factory(latencyCacheSize)
		name := c.Name()
		_, hasGetOrSet := c.(cache.GetOrSetCache)
		c.Close()

		getResult := testing.Benchmark(func(b *testing.B) {
			benchGet(b, factory, keys)
		})
		setResult := testing.Benchmark(func(b *testing.B) {
			benchSet(b, factory, keys)
		})
		setEvictResult := testing.Benchmark(func(b *testing.B) {
			benchSetEvict(b, factory, evictKeys)
		})

		result := LatencyResult{
			Name:           name,
			GetNsOp:        float64(getResult.NsPerOp()),
			SetNsOp:        float64(setResult.NsPerOp()),
			SetEvictNsOp:   float64(setEvictResult.NsPerOp()),
			GetAllocs:      getResult.AllocsPerOp(),
			SetAllocs:      setResult.AllocsPerOp(),
			SetEvictAllocs: setEvictResult.AllocsPerOp(),
			HasGetOrSet:    hasGetOrSet,
		}

		if hasGetOrSet {
			getOrSetResult := testing.Benchmark(func(b *testing.B) {
				benchGetOrSet(b, factory, keys)
			})
			result.GetOrSetNsOp = float64(getOrSetResult.NsPerOp())
			result.GetOrSetAllocs = getOrSetResult.AllocsPerOp()
		}

		results = append(results, result)
	}

	return results
}

// RunIntLatency benchmarks single-threaded Get/Set latency for int-keyed caches.
func RunIntLatency() []IntLatencyResult {
	results := make([]IntLatencyResult, 0, len(cache.AllInt()))

	// Pre-generate keys once for all benchmarks
	keys := make([]int, latencyCacheSize)
	for i := range latencyCacheSize {
		keys[i] = i
	}
	evictKeys := make([]int, latencyCacheSize*20)
	for i := range len(evictKeys) {
		evictKeys[i] = i
	}

	for _, factory := range cache.AllInt() {
		c := factory(latencyCacheSize)
		name := c.Name()
		_, hasGetOrSet := c.(cache.IntGetOrSetCache)
		c.Close()

		getResult := testing.Benchmark(func(b *testing.B) {
			benchIntGet(b, factory, keys)
		})
		setResult := testing.Benchmark(func(b *testing.B) {
			benchIntSet(b, factory, keys)
		})
		setEvictResult := testing.Benchmark(func(b *testing.B) {
			benchIntSetEvict(b, factory, evictKeys)
		})

		result := IntLatencyResult{
			Name:           name,
			GetNsOp:        float64(getResult.NsPerOp()),
			SetNsOp:        float64(setResult.NsPerOp()),
			SetEvictNsOp:   float64(setEvictResult.NsPerOp()),
			GetAllocs:      getResult.AllocsPerOp(),
			SetAllocs:      setResult.AllocsPerOp(),
			SetEvictAllocs: setEvictResult.AllocsPerOp(),
			HasGetOrSet:    hasGetOrSet,
		}

		if hasGetOrSet {
			getOrSetResult := testing.Benchmark(func(b *testing.B) {
				benchIntGetOrSet(b, factory, keys)
			})
			result.GetOrSetNsOp = float64(getOrSetResult.NsPerOp())
			result.GetOrSetAllocs = getOrSetResult.AllocsPerOp()
		}

		results = append(results, result)
	}

	return results
}

func benchGet(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	for _, k := range keys {
		c.Set(k, k)
	}

	b.ResetTimer()
	for i := range b.N {
		c.Get(keys[i%latencyCacheSize])
	}
}

func benchSet(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	b.ResetTimer()
	for i := range b.N {
		c.Set(keys[i%latencyCacheSize], keys[i%latencyCacheSize])
	}
}

func benchSetEvict(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	keySpace := len(keys)
	b.ResetTimer()
	for i := range b.N {
		c.Set(keys[i%keySpace], keys[i%keySpace])
	}
}

func benchIntGet(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	for _, k := range keys {
		c.Set(k, k)
	}

	b.ResetTimer()
	for i := range b.N {
		c.Get(keys[i%latencyCacheSize])
	}
}

func benchIntSet(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	b.ResetTimer()
	for i := range b.N {
		c.Set(keys[i%latencyCacheSize], keys[i%latencyCacheSize])
	}
}

func benchIntSetEvict(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	keySpace := len(keys)
	b.ResetTimer()
	for i := range b.N {
		c.Set(keys[i%keySpace], keys[i%keySpace])
	}
}

func benchGetOrSet(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	gosCache, ok := c.(cache.GetOrSetCache)
	if !ok {
		b.Skip("cache does not implement GetOrSet")
	}

	// Pre-populate half the keys to test both hit and miss cases
	for i := 0; i < latencyCacheSize/2; i++ {
		gosCache.Set(keys[i], keys[i])
	}

	b.ResetTimer()
	for i := range b.N {
		gosCache.GetOrSet(keys[i%latencyCacheSize], keys[i%latencyCacheSize])
	}
}

func benchIntGetOrSet(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	gosCache, ok := c.(cache.IntGetOrSetCache)
	if !ok {
		b.Skip("cache does not implement GetOrSet")
	}

	// Pre-populate half the keys to test both hit and miss cases
	for i := 0; i < latencyCacheSize/2; i++ {
		gosCache.Set(keys[i], keys[i])
	}

	b.ResetTimer()
	for i := range b.N {
		gosCache.GetOrSet(keys[i%latencyCacheSize], keys[i%latencyCacheSize])
	}
}
