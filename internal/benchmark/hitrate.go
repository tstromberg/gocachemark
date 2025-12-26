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
	Rates map[int]float64
	Name  string
}

// DefaultCacheSizes are the cache sizes to benchmark.
var DefaultCacheSizes = []int{16_384, 32_768, 65_536, 131_072, 262_144}

// Entry size constants for different workloads (key + value + overhead).
// These ensure byte-based caches like freecache are sized fairly.
const (
	CDNEntrySize          = 190 // avg key ~77 bytes, value=key, ~32 bytes overhead
	MetaEntrySize         = 55  // avg key ~10 bytes, value=key, ~32 bytes overhead
	ZipfEntrySize         = 45  // key ~6 bytes (int as string), value=key, ~32 bytes overhead
	TwitterEntrySize      = 110 // avg key ~40 bytes, value=key, ~32 bytes overhead
	WikipediaEntrySize    = 55  // avg key ~10 bytes, value=key, ~32 bytes overhead
	ThesiosBlockEntrySize = 180 // key ~72 bytes (hash:offset), value=key, ~32 bytes overhead
	ThesiosFileEntrySize  = 160 // key 64 bytes (hash only), value=key, ~32 bytes overhead
	IBMDockerEntrySize    = 115 // key ~40 bytes (URI), value=key, ~32 bytes overhead
	TencentPhotoEntrySize = 115 // key 40 bytes (hex hash), value=key, ~32 bytes overhead
)

// runStringHitRate is the common implementation for string-based trace benchmarks.
func runStringHitRate(loader func() ([]string, error), entrySize int, sizes []int, traceName string) ([]HitRateResult, error) {
	ops, err := loader()
	if err != nil {
		return nil, fmt.Errorf("load %s trace: %w", traceName, err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(entrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runStringTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

// RunCDNHitRate benchmarks hit rates using the CDN production trace.
func RunCDNHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadCDNTrace, CDNEntrySize, sizes, "CDN")
}

// runStringTrace runs a hit rate benchmark using string keys.
// Used by CDN, Twitter, Wikipedia, Thesios, IBM Docker, and Tencent Photo traces.
func runStringTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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

func runMetaTrace(factory cache.Factory, ops []trace.Op, cacheSize int) float64 {
	c := factory(cacheSize)
	defer c.Close()

	var hits, misses int64
	for _, op := range ops {
		switch op.Action {
		case "GET":
			if _, ok := c.Get(op.Key); ok {
				hits++
			} else {
				misses++
				c.Set(op.Key, op.Key)
			}
		case "SET":
			c.Set(op.Key, op.Key)
		default:
			// ignore unknown operations
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

// RunTwitterHitRate benchmarks hit rates using the Twitter production trace.
func RunTwitterHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadTwitterTrace, TwitterEntrySize, sizes, "Twitter")
}

// RunWikipediaHitRate benchmarks hit rates using the Wikipedia CDN upload trace.
func RunWikipediaHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadWikipediaTrace, WikipediaEntrySize, sizes, "Wikipedia")
}

// RunThesiosBlockHitRate benchmarks hit rates using the Google Thesios I/O block trace.
func RunThesiosBlockHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadThesiosBlockTrace, ThesiosBlockEntrySize, sizes, "Thesios block")
}

// RunThesiosFileHitRate benchmarks hit rates using the Google Thesios I/O file trace.
func RunThesiosFileHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadThesiosFileTrace, ThesiosFileEntrySize, sizes, "Thesios file")
}

// RunIBMDockerHitRate benchmarks hit rates using the IBM Docker Registry trace.
func RunIBMDockerHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadIBMDockerTrace, IBMDockerEntrySize, sizes, "IBM Docker")
}

// RunTencentPhotoHitRate benchmarks hit rates using the Tencent Photo trace.
func RunTencentPhotoHitRate(sizes []int) ([]HitRateResult, error) {
	return runStringHitRate(trace.LoadTencentPhotoTrace, TencentPhotoEntrySize, sizes, "Tencent Photo")
}
