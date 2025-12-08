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
	throughputAlpha        = 0.99
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
	wg.Wait()

	return float64(ops.Load()) / benchmarkDuration.Seconds()
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
	wg.Wait()

	return float64(ops.Load()) / benchmarkDuration.Seconds()
}
