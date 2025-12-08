package benchmark

import (
	"strconv"
	"testing"

	"github.com/tstromberg/gocachemark/internal/cache"
)

// LatencyResult holds single-threaded latency results for a cache.
type LatencyResult struct {
	Name           string
	GetNsOp        float64 // nanoseconds per Get operation
	SetNsOp        float64 // nanoseconds per Set operation (no eviction)
	SetEvictNsOp   float64 // nanoseconds per Set with eviction (20x keyspace)
	GetAllocs      int64   // allocations per Get
	SetAllocs      int64   // allocations per Set
	SetEvictAllocs int64   // allocations per Set with eviction
}

// IntLatencyResult holds single-threaded latency results for int-keyed caches.
type IntLatencyResult struct {
	Name      string
	GetNsOp   float64
	SetNsOp   float64
	GetAllocs int64
	SetAllocs int64
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

		results = append(results, LatencyResult{
			Name:           name,
			GetNsOp:        float64(getResult.NsPerOp()),
			SetNsOp:        float64(setResult.NsPerOp()),
			SetEvictNsOp:   float64(setEvictResult.NsPerOp()),
			GetAllocs:      getResult.AllocsPerOp(),
			SetAllocs:      setResult.AllocsPerOp(),
			SetEvictAllocs: setEvictResult.AllocsPerOp(),
		})
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

	for _, factory := range cache.AllInt() {
		c := factory(latencyCacheSize)
		name := c.Name()
		c.Close()

		getResult := testing.Benchmark(func(b *testing.B) {
			benchIntGet(b, factory, keys)
		})
		setResult := testing.Benchmark(func(b *testing.B) {
			benchIntSet(b, factory, keys)
		})

		results = append(results, IntLatencyResult{
			Name:      name,
			GetNsOp:   float64(getResult.NsPerOp()),
			SetNsOp:   float64(setResult.NsPerOp()),
			GetAllocs: getResult.AllocsPerOp(),
			SetAllocs: setResult.AllocsPerOp(),
		})
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
