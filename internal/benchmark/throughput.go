package benchmark

import (
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
	throughputCacheSize    = 10000
	throughputWorkloadSize = 1_000_000
	throughputAlpha        = 0.8
	benchmarkDuration      = 1 * time.Second
	opsBatchSize           = 1000
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

	// Pre-populate cache
	for i := range throughputCacheSize {
		c.Set(strconv.Itoa(i), strconv.Itoa(i))
	}

	// Pre-convert keys to strings
	keyStrs := make([]string, len(keys))
	for i, k := range keys {
		keyStrs[i] = strconv.Itoa(k)
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
					key := keyStrs[i%workloadLen]
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

// RunGetOrSetThroughput benchmarks GetOrSet throughput at various thread counts (string keys).
func RunGetOrSetThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, throughputCacheSize, throughputAlpha, 42)

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

// RunIntGetOrSetThroughput benchmarks GetOrSet throughput at various thread counts (int keys).
func RunIntGetOrSetThroughput(threadCounts []int) []ThroughputResult {
	keys := workload.GenerateZipfInt(throughputWorkloadSize, throughputCacheSize, throughputAlpha, 42)

	results := make([]ThroughputResult, 0)
	for _, factory := range cache.AllInt() {
		c := factory(throughputCacheSize)
		_, hasGetOrSet := c.(cache.IntGetOrSetCache)
		name := c.Name()
		c.Close()

		if !hasGetOrSet {
			continue
		}

		qps := make(map[int]float64)
		for _, threads := range threadCounts {
			qps[threads] = measureIntGetOrSetQPS(factory, keys, threads)
		}
		results = append(results, ThroughputResult{Name: name, QPS: qps})
	}

	return results
}

func measureGetOrSetQPS(factory cache.Factory, keys []int, threads int) float64 {
	c := factory(throughputCacheSize)
	defer c.Close()

	gosCache, ok := c.(cache.GetOrSetCache)
	if !ok {
		return 0
	}

	// Pre-populate half the cache to test both hit and miss cases
	for i := 0; i < throughputCacheSize/2; i++ {
		gosCache.Set(strconv.Itoa(i), strconv.Itoa(i))
	}

	// Pre-convert keys to strings
	keyStrs := make([]string, len(keys))
	for i, k := range keys {
		keyStrs[i] = strconv.Itoa(k)
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
					key := keyStrs[i%workloadLen]
					gosCache.GetOrSet(key, key)
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

func measureIntGetOrSetQPS(factory cache.IntFactory, keys []int, threads int) float64 {
	c := factory(throughputCacheSize)
	defer c.Close()

	gosCache, ok := c.(cache.IntGetOrSetCache)
	if !ok {
		return 0
	}

	// Pre-populate half the cache to test both hit and miss cases
	for i := 0; i < throughputCacheSize/2; i++ {
		gosCache.Set(i, i)
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
					gosCache.GetOrSet(key, key)
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
