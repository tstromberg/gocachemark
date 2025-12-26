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
	ThroughputCacheSize    = 65536 // 64K - realistic cache size for multi-threaded benchmarks
	throughputWorkloadSize = 1_000_000
	throughputAlpha        = 0.8
	benchmarkDuration      = 900 * time.Millisecond
	opsBatchSize           = 1000
	throughputValueSize    = 4 * 1024  // 4KB for regular throughput
	getOrSetValueSize      = 8 * 1024  // 8KB for GetOrSet
)

// RunGetOrSetThroughput benchmarks GetOrSet throughput at various thread counts.
// Uses URL-like keys for realistic cache workloads.
func RunGetOrSetThroughput(threadCounts []int) []ThroughputResult {
	keys := generateThroughputURLKeys(throughputWorkloadSize)

	results := make([]ThroughputResult, 0)
	for _, factory := range cache.All() {
		c := factory(ThroughputCacheSize)
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
	indices := workload.GenerateZipfInt(n, ThroughputCacheSize, throughputAlpha, 42)

	keys := make([]string, n)
	for i, idx := range indices {
		seg := segments[idx%len(segments)]
		keys[i] = fmt.Sprintf("https://en.wikipedia.org/wiki/%s_%d", seg, idx)
	}
	return keys
}

func measureGetOrSetQPS(factory cache.Factory, keys []string, threads int) float64 {
	c := factory(ThroughputCacheSize)
	defer c.Close()

	gosCache, ok := c.(cache.GetOrSetCache)
	if !ok {
		return 0
	}

	// Generate 8KB values
	values := make([]string, ThroughputCacheSize)
	baseValue := make([]byte, getOrSetValueSize)
	for i := range baseValue {
		baseValue[i] = byte('A' + (i % 26))
	}
	for i := range ThroughputCacheSize {
		values[i] = string(baseValue)
	}

	// Pre-populate half the cache to test both hit and miss cases
	for i := 0; i < len(keys)/2 && i < ThroughputCacheSize; i++ {
		gosCache.Set(keys[i], values[i%ThroughputCacheSize])
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
					gosCache.GetOrSet(key, values[i%ThroughputCacheSize])
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

// RunStringGetThroughput benchmarks Get throughput with string keys.
func RunStringGetThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, ThroughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0, len(cache.All()))
	for _, factory := range cache.All() {
		c := factory(ThroughputCacheSize)
		name := c.Name()
		c.Close()

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureStringGetQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

// RunStringSetThroughput benchmarks Set throughput with string keys.
func RunStringSetThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, ThroughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0, len(cache.All()))
	for _, factory := range cache.All() {
		c := factory(ThroughputCacheSize)
		name := c.Name()
		c.Close()

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureStringSetQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

// RunIntGetThroughput benchmarks Get throughput with int keys.
func RunIntGetThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, ThroughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0, len(cache.AllInt()))
	for _, factory := range cache.AllInt() {
		c := factory(ThroughputCacheSize)
		name := c.Name()
		c.Close()

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureIntGetQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

// RunIntSetThroughput benchmarks Set throughput with int keys.
func RunIntSetThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, ThroughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0, len(cache.AllInt()))
	for _, factory := range cache.AllInt() {
		c := factory(ThroughputCacheSize)
		name := c.Name()
		c.Close()

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureIntSetQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

func measureStringGetQPS(factory cache.Factory, keys []int, threads int) float64 {
	c := factory(ThroughputCacheSize)
	defer c.Close()

	keyStrs := make([]string, ThroughputCacheSize)
	values := make([]string, ThroughputCacheSize)
	baseValue := make([]byte, throughputValueSize)
	for i := range baseValue {
		baseValue[i] = byte('A' + (i % 26))
	}
	for i := range ThroughputCacheSize {
		keyStrs[i] = strconv.Itoa(i)
		values[i] = string(baseValue)
	}

	for i := range ThroughputCacheSize {
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
					c.Get(keyStrs[idx])
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

func measureStringSetQPS(factory cache.Factory, keys []int, threads int) float64 {
	c := factory(ThroughputCacheSize)
	defer c.Close()

	keyStrs := make([]string, ThroughputCacheSize)
	values := make([]string, ThroughputCacheSize)
	baseValue := make([]byte, throughputValueSize)
	for i := range baseValue {
		baseValue[i] = byte('A' + (i % 26))
	}
	for i := range ThroughputCacheSize {
		keyStrs[i] = strconv.Itoa(i)
		values[i] = string(baseValue)
	}

	for i := range ThroughputCacheSize {
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
					c.Set(keyStrs[idx], values[idx])
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

func measureIntGetQPS(factory cache.IntFactory, keys []int, threads int) float64 {
	c := factory(ThroughputCacheSize)
	defer c.Close()

	for i := range ThroughputCacheSize {
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
					c.Get(keys[i%workloadLen])
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

func measureIntSetQPS(factory cache.IntFactory, keys []int, threads int) float64 {
	c := factory(ThroughputCacheSize)
	defer c.Close()

	for i := range ThroughputCacheSize {
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
					c.Set(key, key)
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
