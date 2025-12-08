// Package benchmark implements cache benchmark runners.
package benchmark

import (
	"fmt"
	"strconv"

	"github.com/tstromberg/gocachemark/internal/cache"
	"github.com/tstromberg/gocachemark/internal/trace"
	"github.com/tstromberg/gocachemark/internal/workload"
)

// HitRateResult holds hit rate results for a single cache.
type HitRateResult struct {
	Name  string
	Rates map[int]float64 // cache size -> hit rate percentage
}

// DefaultCacheSizes are the cache sizes to benchmark.
var DefaultCacheSizes = []int{16_384, 32_768, 65_536, 131_072, 262_144}

// Entry size constants for different workloads (key + value + overhead).
// These ensure byte-based caches like freecache are sized fairly.
const (
	// CDN trace: avg key ~77 bytes, value=key, ~32 bytes overhead
	CDNEntrySize = 190
	// Meta trace: avg key ~10 bytes, value=key, ~32 bytes overhead
	MetaEntrySize = 55
	// Zipf trace: key ~6 bytes (int as string), value=key, ~32 bytes overhead
	ZipfEntrySize = 45
)

// RunCDNHitRate benchmarks hit rates using the CDN production trace.
func RunCDNHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadCDNTrace()
	if err != nil {
		return nil, fmt.Errorf("load CDN trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(CDNEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runCDNTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runCDNTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
	c := factory(cacheSize)
	defer c.Close()

	var hits, misses int64
	for _, key := range ops {
		if _, ok := c.Get(key); ok {
			hits++
		} else {
			misses++
			c.Set(key, key)
		}
	}
	return float64(hits) / float64(hits+misses) * 100
}

// RunMetaHitRate benchmarks hit rates using the Meta KVCache production trace.
func RunMetaHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadMetaTrace()
	if err != nil {
		return nil, fmt.Errorf("load Meta trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(MetaEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runMetaTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runMetaTrace(factory cache.Factory, ops []trace.TraceOp, cacheSize int) float64 {
	c := factory(cacheSize)
	defer c.Close()

	var hits, misses int64
	for _, op := range ops {
		switch op.Op {
		case "GET":
			if _, ok := c.Get(op.Key); ok {
				hits++
			} else {
				misses++
				c.Set(op.Key, op.Key)
			}
		case "SET":
			c.Set(op.Key, op.Key)
		}
	}
	return float64(hits) / float64(hits+misses) * 100
}

// RunZipfHitRate benchmarks hit rates using synthetic Zipf distribution.
func RunZipfHitRate(sizes []int, keySpace, workloadSize int, alpha float64) []HitRateResult {
	keys := workload.GenerateZipfInt(workloadSize, keySpace, alpha, 42)

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(ZipfEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runZipfTrace(factory, keys, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results
}

func runZipfTrace(factory cache.Factory, keys []int, cacheSize int) float64 {
	c := factory(cacheSize)
	defer c.Close()

	var hits int64
	for _, key := range keys {
		k := strconv.Itoa(key)
		if _, ok := c.Get(k); ok {
			hits++
		} else {
			c.Set(k, k)
		}
	}
	return float64(hits) / float64(len(keys)) * 100
}
