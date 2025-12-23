package benchmark

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tstromberg/gocachemark/internal/cache"
	"github.com/tstromberg/gocachemark/internal/workload"
)

// ThroughputResult holds multi-threaded throughput results for a cache.
type ThroughputResult struct {
	Name string
	QPS  map[int]float64 // thread count -> QPS
}

// DefaultThreadCounts are the thread counts to benchmark.
var DefaultThreadCounts = []int{1, 8, 16, 32}

const (
	throughputCacheSize    = 32768 // 32K - realistic cache size for multi-threaded benchmarks
	throughputWorkloadSize = 1_000_000
	throughputAlpha        = 0.8
	benchmarkDuration      = 1 * time.Second
	opsBatchSize           = 1000
	throughputValueSize    = 4 * 1024  // 4KB for regular throughput
	getOrSetValueSize      = 8 * 1024  // 8KB for GetOrSet
)

// RunThroughput benchmarks throughput at various thread counts using Zipf workload (string keys).
// Uses 75% reads / 25% writes pattern.
func RunThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, throughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0, len(cache.All()))
	for _, factory := range cache.All() {
		c := factory(throughputCacheSize)
		name := c.Name()
		c.Close()

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

// RunIntThroughput benchmarks throughput at various thread counts using Zipf workload (int keys).
// Uses 75% reads / 25% writes pattern.
func RunIntThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, throughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0, len(cache.AllInt()))
	for _, factory := range cache.AllInt() {
		c := factory(throughputCacheSize)
		name := c.Name()
		c.Close()

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureIntQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

func measureQPS(factory cache.Factory, keys []int, threads int) float64 {
	c := factory(throughputCacheSize)
	defer c.Close()

	// Pre-convert keys to strings and generate 4KB values
	keyStrs := make([]string, throughputCacheSize)
	values := make([]string, throughputCacheSize)
	baseValue := make([]byte, throughputValueSize)
	for i := range baseValue {
		baseValue[i] = byte('A' + (i % 26))
	}
	for i := range throughputCacheSize {
		keyStrs[i] = strconv.Itoa(i)
		values[i] = string(baseValue)
	}

	// Pre-populate cache
	for i := range throughputCacheSize {
		c.Set(keyStrs[i], values[i])
	}

	var ops atomic.Int64
	var stop atomic.Bool
	var wg sync.WaitGroup

	workloadLen := len(keys)

	for range threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; {
				for range opsBatchSize {
					idx := keys[i%workloadLen]
					key := keyStrs[idx]
					if i%4 == 0 { // 25% writes
						c.Set(key, values[idx])
					} else { // 75% reads
						c.Get(key)
					}
					i++
				}
				ops.Add(opsBatchSize)
				if stop.Load() {
					return
				}
			}
		}()
	}

	time.Sleep(benchmarkDuration)
	stop.Store(true)

	// Wait with timeout to detect hangs
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return float64(ops.Load()) / benchmarkDuration.Seconds()
	case <-time.After(5 * time.Second):
		// Benchmark hung - return 0 to indicate failure
		return 0
	}
}

func measureIntQPS(factory cache.IntFactory, keys []int, threads int) float64 {
	c := factory(throughputCacheSize)
	defer c.Close()

	// Pre-populate cache
	for i := range throughputCacheSize {
		c.Set(i, i)
	}

	var ops atomic.Int64
	var stop atomic.Bool
	var wg sync.WaitGroup

	workloadLen := len(keys)

	for range threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; {
				for range opsBatchSize {
					key := keys[i%workloadLen]
					if i%4 == 0 { // 25% writes
						c.Set(key, key)
					} else { // 75% reads
						c.Get(key)
					}
					i++
				}
				ops.Add(opsBatchSize)
				if stop.Load() {
					return
				}
			}
		}()
	}

	time.Sleep(benchmarkDuration)
	stop.Store(true)

	// Wait with timeout to detect hangs
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return float64(ops.Load()) / benchmarkDuration.Seconds()
	case <-time.After(5 * time.Second):
		// Benchmark hung - return 0 to indicate failure
		return 0
	}
}

// RunGetOrSetThroughput benchmarks GetOrSet throughput at various thread counts.
// Uses URL-like keys for realistic cache workloads.
func RunGetOrSetThroughput(threadCounts []int) []ThroughputResult {
	keys := generateThroughputURLKeys(throughputWorkloadSize)

	results := make([]ThroughputResult, 0)
	for _, factory := range cache.All() {
		c := factory(throughputCacheSize)
		_, hasGetOrSet := c.(cache.GetOrSetCache)
		name := c.Name()
		c.Close()

		if !hasGetOrSet {
			continue
		}

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureGetOrSetQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

// generateThroughputURLKeys creates URL-like cache keys with Zipf distribution.
func generateThroughputURLKeys(n int) []string {
	segments := []string{
		"Main_Page", "United_States", "World_War_II", "India", "United_Kingdom",
		"Canada", "Australia", "Germany", "France", "Japan", "China", "Russia",
		"Brazil", "Italy", "Spain", "Mexico", "South_Korea", "Indonesia",
		"New_York_City", "London", "Paris", "Tokyo", "Los_Angeles", "Chicago",
	}

	// Generate Zipf-distributed indices
	indices := workload.GenerateZipfInt(n, throughputCacheSize, throughputAlpha, 42)

	keys := make([]string, n)
	for i, idx := range indices {
		seg := segments[idx%len(segments)]
		keys[i] = fmt.Sprintf("https://en.wikipedia.org/wiki/%s_%d", seg, idx)
	}
	return keys
}

func measureGetOrSetQPS(factory cache.Factory, keys []string, threads int) float64 {
	c := factory(throughputCacheSize)
	defer c.Close()

	gosCache, ok := c.(cache.GetOrSetCache)
	if !ok {
		return 0
	}

	// Generate 8KB values
	values := make([]string, throughputCacheSize)
	baseValue := make([]byte, getOrSetValueSize)
	for i := range baseValue {
		baseValue[i] = byte('A' + (i % 26))
	}
	for i := range throughputCacheSize {
		values[i] = string(baseValue)
	}

	// Pre-populate half the cache to test both hit and miss cases
	for i := 0; i < len(keys)/2 && i < throughputCacheSize; i++ {
		gosCache.Set(keys[i], values[i%throughputCacheSize])
	}

	var ops atomic.Int64
	var stop atomic.Bool
	var wg sync.WaitGroup

	workloadLen := len(keys)

	for range threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; {
				for range opsBatchSize {
					key := keys[i%workloadLen]
					gosCache.GetOrSet(key, values[i%throughputCacheSize])
					i++
				}
				ops.Add(opsBatchSize)
				if stop.Load() {
					return
				}
			}
		}()
	}

	time.Sleep(benchmarkDuration)
	stop.Store(true)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return float64(ops.Load()) / benchmarkDuration.Seconds()
	case <-time.After(5 * time.Second):
		return 0
	}
}
