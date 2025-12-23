package benchmark

import (
	"fmt"
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
	Name           string
	GetNsOp        float64
	SetNsOp        float64
	SetEvictNsOp   float64
	GetAllocs      int64
	SetAllocs      int64
	SetEvictAllocs int64
}

// GetOrSetLatencyResult holds single-threaded GetOrSet latency results.
type GetOrSetLatencyResult struct {
	Name   string
	NsOp   float64
	Allocs int64
}

const latencyCacheSize = 16384 // Power of 2 for better hash distribution

// generateURLKeys creates URL-like cache keys for realistic benchmarking.
// Uses a fixed seed for reproducibility.
func generateURLKeys(n int) []string {
	// Common Wikipedia article path segments for realistic URLs
	segments := []string{
		"Main_Page", "United_States", "World_War_II", "India", "United_Kingdom",
		"Canada", "Australia", "Germany", "France", "Japan", "China", "Russia",
		"Brazil", "Italy", "Spain", "Mexico", "South_Korea", "Indonesia",
		"New_York_City", "London", "Paris", "Tokyo", "Los_Angeles", "Chicago",
		"Houston", "Phoenix", "Philadelphia", "San_Antonio", "San_Diego",
		"Albert_Einstein", "Isaac_Newton", "Charles_Darwin", "Marie_Curie",
		"Leonardo_da_Vinci", "William_Shakespeare", "Abraham_Lincoln",
		"George_Washington", "Napoleon", "Julius_Caesar", "Cleopatra",
		"The_Beatles", "Elvis_Presley", "Michael_Jackson", "Madonna",
		"Python_(programming_language)", "JavaScript", "Java_(programming_language)",
		"C_(programming_language)", "Go_(programming_language)", "Rust_(programming_language)",
		"Linux", "Microsoft_Windows", "MacOS", "Android_(operating_system)", "IOS",
	}

	keys := make([]string, n)
	for i := range n {
		// Deterministic selection based on index
		seg := segments[i%len(segments)]
		keys[i] = fmt.Sprintf("https://en.wikipedia.org/wiki/%s_%d", seg, i)
	}
	return keys
}

// RunLatency benchmarks single-threaded Get/Set latency for all caches (string keys).
func RunLatency() []LatencyResult {
	results := make([]LatencyResult, 0, len(cache.All()))

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

// RunGetOrSetLatency benchmarks single-threaded GetOrSet latency for caches that support it.
// Uses URL-like keys to simulate realistic cache workloads.
func RunGetOrSetLatency() []GetOrSetLatencyResult {
	var results []GetOrSetLatencyResult

	keys := generateURLKeys(latencyCacheSize)

	for _, factory := range cache.All() {
		c := factory(latencyCacheSize)
		name := c.Name()
		_, hasGetOrSet := c.(cache.GetOrSetCache)
		c.Close()

		if !hasGetOrSet {
			continue
		}

		result := testing.Benchmark(func(b *testing.B) {
			benchGetOrSet(b, factory, keys)
		})

		results = append(results, GetOrSetLatencyResult{
			Name:   name,
			NsOp:   float64(result.NsPerOp()),
			Allocs: result.AllocsPerOp(),
		})
	}

	return results
}

// RunIntLatency benchmarks single-threaded Get/Set latency for int-keyed caches.
func RunIntLatency() []IntLatencyResult {
	results := make([]IntLatencyResult, 0, len(cache.AllInt()))

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

		results = append(results, IntLatencyResult{
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

func benchGet(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	for _, k := range keys {
		c.Set(k, k)
	}

	n := len(keys)
	b.ResetTimer()
	for i := range b.N {
		c.Get(keys[i%n])
	}
}

func benchSet(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	// Pre-populate to ensure we're measuring updates, not inserts.
	for _, k := range keys {
		c.Set(k, k)
	}

	n := len(keys)
	b.ResetTimer()
	for i := range b.N {
		k := keys[i%n]
		c.Set(k, k)
	}
}

func benchSetEvict(b *testing.B, factory cache.Factory, keys []string) {
	c := factory(latencyCacheSize)
	defer c.Close()

	// Pre-fill cache to capacity, then measure eviction with new keys.
	for i := 0; i < latencyCacheSize; i++ {
		c.Set(keys[i], keys[i])
	}

	// Use keys beyond the pre-filled set to force eviction on every Set.
	evictKeys := keys[latencyCacheSize:]
	n := len(evictKeys)
	b.ResetTimer()
	for i := range b.N {
		k := evictKeys[i%n]
		c.Set(k, k)
	}
}

func benchIntGet(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	for _, k := range keys {
		c.Set(k, k)
	}

	n := len(keys)
	b.ResetTimer()
	for i := range b.N {
		c.Get(keys[i%n])
	}
}

func benchIntSet(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	// Pre-populate to ensure we're measuring updates, not inserts.
	for _, k := range keys {
		c.Set(k, k)
	}

	n := len(keys)
	b.ResetTimer()
	for i := range b.N {
		k := keys[i%n]
		c.Set(k, k)
	}
}

func benchIntSetEvict(b *testing.B, factory cache.IntFactory, keys []int) {
	c := factory(latencyCacheSize)
	defer c.Close()

	// Pre-fill cache to capacity.
	for i := 0; i < latencyCacheSize; i++ {
		c.Set(keys[i], keys[i])
	}

	// Use keys beyond the pre-filled set to force eviction.
	evictKeys := keys[latencyCacheSize:]
	n := len(evictKeys)
	b.ResetTimer()
	for i := range b.N {
		k := evictKeys[i%n]
		c.Set(k, k)
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

	n := len(keys)
	b.ResetTimer()
	for i := range b.N {
		k := keys[i%n]
		gosCache.GetOrSet(k, k)
	}
}

