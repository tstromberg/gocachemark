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
	// Twitter trace: avg key ~40 bytes, value=key, ~32 bytes overhead
	TwitterEntrySize = 110
	// Wikipedia trace: avg key ~10 bytes, value=key, ~32 bytes overhead
	WikipediaEntrySize = 55
	// Thesios block trace: key is ~72 bytes (hash:offset), value=key, ~32 bytes overhead
	ThesiosBlockEntrySize = 180
	// Thesios file trace: key is 64 bytes (hash only), value=key, ~32 bytes overhead
	ThesiosFileEntrySize = 160
	// IBM Docker trace: key is ~40 bytes (URI), value=key, ~32 bytes overhead
	IBMDockerEntrySize = 115
	// Tencent Photo trace: key is 40 bytes (hex hash), value=key, ~32 bytes overhead
	TencentPhotoEntrySize = 115
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

// RunTwitterHitRate benchmarks hit rates using the Twitter production trace.
func RunTwitterHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadTwitterTrace()
	if err != nil {
		return nil, fmt.Errorf("load Twitter trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(TwitterEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runTwitterTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runTwitterTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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

// RunWikipediaHitRate benchmarks hit rates using the Wikipedia CDN upload trace.
func RunWikipediaHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadWikipediaTrace()
	if err != nil {
		return nil, fmt.Errorf("load Wikipedia trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(WikipediaEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runWikipediaTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runWikipediaTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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

// RunThesiosBlockHitRate benchmarks hit rates using the Google Thesios I/O block trace.
func RunThesiosBlockHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadThesiosBlockTrace()
	if err != nil {
		return nil, fmt.Errorf("load Thesios block trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(ThesiosBlockEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runThesiosBlockTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runThesiosBlockTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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

// RunThesiosFileHitRate benchmarks hit rates using the Google Thesios I/O file trace.
func RunThesiosFileHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadThesiosFileTrace()
	if err != nil {
		return nil, fmt.Errorf("load Thesios file trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(ThesiosFileEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runThesiosFileTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runThesiosFileTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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

// RunIBMDockerHitRate benchmarks hit rates using the IBM Docker Registry trace.
func RunIBMDockerHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadIBMDockerTrace()
	if err != nil {
		return nil, fmt.Errorf("load IBM Docker trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(IBMDockerEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runIBMDockerTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runIBMDockerTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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

// RunTencentPhotoHitRate benchmarks hit rates using the Tencent Photo trace.
func RunTencentPhotoHitRate(sizes []int) ([]HitRateResult, error) {
	ops, err := trace.LoadTencentPhotoTrace()
	if err != nil {
		return nil, fmt.Errorf("load Tencent Photo trace: %w", err)
	}

	results := make([]HitRateResult, 0, len(cache.All()))
	for _, factory := range cache.AllWithEntrySize(TencentPhotoEntrySize) {
		c := factory(sizes[0])
		name := c.Name()
		c.Close()

		rates := make(map[int]float64)
		for _, size := range sizes {
			rates[size] = runTencentPhotoTrace(factory, ops, size)
		}
		results = append(results, HitRateResult{Name: name, Rates: rates})
	}

	return results, nil
}

func runTencentPhotoTrace(factory cache.Factory, ops []string, cacheSize int) float64 {
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
