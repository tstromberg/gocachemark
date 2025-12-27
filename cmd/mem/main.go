// Package main measures memory usage for a single cache implementation.
// Run in isolated process for accurate measurements.
package main

import (
	"flag"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/Code-Hex/go-generics-cache/policy/clock"
	"github.com/Yiling-J/theine-go"
	"github.com/codeGROOVE-dev/multicache"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	"github.com/dgryski/go-s4lru"
	"github.com/elastic/go-freelru"
	lru2 "github.com/hashicorp/golang-lru/v2"
	"github.com/jellydator/ttlcache/v3"
	"github.com/maypok86/otter/v2"
	"github.com/scalalang2/golang-fifo/s3fifo"
	"github.com/scalalang2/golang-fifo/sieve"
	tinylfu "github.com/vmihailenco/go-tinylfu"
	"github.com/zeebo/xxh3"
)

func main() {
	cacheName := flag.String("cache", "", "cache implementation to benchmark")
	capacity := flag.Int("cap", 32768, "capacity")
	valSize := flag.Int("valSize", 1024, "value size in bytes")
	flag.Parse()

	if *cacheName == "" {
		fmt.Println(`{"error":"cache name required"}`)
		return
	}

	runtime.GC() //nolint:revive // intentional for memory measurement
	debug.FreeOSMemory()

	items, data := runBenchmark(*cacheName, *capacity, *valSize)

	runtime.GC() //nolint:revive // intentional for memory measurement
	time.Sleep(100 * time.Millisecond)
	runtime.GC() //nolint:revive // intentional for memory measurement
	debug.FreeOSMemory()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// Keep data alive until after measurement
	runtime.KeepAlive(data)

	fmt.Printf(`{"name":%q, "items":%d, "bytes":%d}`, *cacheName, items, mem.Alloc)
}

//nolint:gocritic // unnamed results and eval order are intentional for memory measurement
func runBenchmark(name string, capacity, valSize int) (int, any) {
	switch name {
	case "baseline":
		return runBaseline(capacity, valSize)
	case "multicache":
		return runMulticache(capacity, valSize)
	case "otter":
		return runOtter(capacity, valSize)
	case "theine":
		return runTheine(capacity, valSize)
	case "ristretto":
		return runRistretto(capacity, valSize)
	case "tinylfu":
		return runTinyLFU(capacity, valSize)
	case "sieve":
		return runSieve(capacity, valSize)
	case "s3-fifo":
		return runS3FIFO(capacity, valSize)
	case "freelru-shard":
		return runFreeLRUSharded(capacity, valSize)
	case "freelru-sync":
		return runFreeLRUSynced(capacity, valSize)
	case "freecache":
		return runFreecache(capacity, valSize)
	case "2q":
		return runTwoQueue(capacity, valSize)
	case "s4lru":
		return runS4LRU(capacity, valSize)
	case "clock":
		return runClock(capacity, valSize)
	case "lru":
		return runLRU(capacity, valSize)
	case "ttlcache":
		return runTTLCache(capacity, valSize)
	default:
		return 0, nil
	}
}

//nolint:gocritic // unnamed results and eval order are intentional
func runBaseline(capacity, valSize int) (int, any) {
	m := make(map[string][]byte, capacity)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		// Write data to force actual memory allocation (avoid zero-page optimization)
		for j := range val {
			val[j] = byte(i + j)
		}
		m[key] = val
	}
	return len(m), m
}

//nolint:gocritic // unnamed results and eval order are intentional
func runMulticache(capacity, valSize int) (int, any) {
	c := multicache.New[string, []byte](multicache.Size(capacity))
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runOtter(capacity, valSize int) (int, any) {
	c := otter.Must(&otter.Options[string, []byte]{MaximumSize: capacity})
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val)
	}
	return c.EstimatedSize(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runTheine(capacity, valSize int) (int, any) {
	c, _ := theine.NewBuilder[string, []byte](int64(capacity)).Build() //nolint:errcheck // capacity always valid
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val, 0)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runRistretto(capacity, valSize int) (int, any) {
	c, _ := ristretto.NewCache(&ristretto.Config{ //nolint:errcheck // capacity always valid
		NumCounters:        int64(capacity * 10),
		MaxCost:            int64(capacity),
		BufferItems:        64,
		IgnoreInternalCost: true,
	})

	// Ristretto uses TinyLFU admission - need multiple passes to build frequency
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			val := make([]byte, valSize)
			for j := range val {
				val[j] = byte(i + j)
			}
			c.Set(key, val, 1)
			if pass > 0 {
				c.Get(key)
			}
		}
		c.Wait()
	}

	count := 0
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		if _, ok := c.Get(key); ok {
			count++
		}
	}
	return count, c
}

type tinyLFUWrapper struct {
	c *tinylfu.T
}

//nolint:gocritic // unnamed results and eval order are intentional
func runTinyLFU(capacity, valSize int) (int, any) {
	// Use non-sync version - SyncT has issues with admission
	c := tinylfu.New(capacity, capacity*10)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(&tinylfu.Item{Key: key, Value: val})
	}
	w := &tinyLFUWrapper{c: c}

	count := 0
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		if _, ok := c.Get(key); ok {
			count++
		}
	}
	return count, w
}

//nolint:gocritic // unnamed results and eval order are intentional
func runSieve(capacity, valSize int) (int, any) {
	c := sieve.New[string, []byte](capacity, 0)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runS3FIFO(capacity, valSize int) (int, any) {
	c := s3fifo.New[string, []byte](capacity, 0)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val)
	}
	return c.Len(), c
}

func hashString(s string) uint32 {
	return uint32(xxh3.HashString(s)) //nolint:gosec // safe truncation
}

//nolint:gocritic // unnamed results and eval order are intentional
func runFreeLRUSharded(capacity, valSize int) (int, any) {
	c, _ := freelru.NewSharded[string, []byte](uint32(capacity), hashString) //nolint:errcheck,gosec // capacity always valid
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Add(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runFreeLRUSynced(capacity, valSize int) (int, any) {
	c, _ := freelru.NewSynced[string, []byte](uint32(capacity), hashString) //nolint:errcheck,gosec // capacity always valid
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Add(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runFreecache(capacity, valSize int) (int, any) {
	overhead := 256
	size := capacity * (valSize + overhead)
	c := freecache.NewCache(size)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set([]byte(key), val, 0) //nolint:errcheck,gosec // best-effort set
	}
	return int(c.EntryCount()), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runTwoQueue(capacity, valSize int) (int, any) {
	c, _ := lru2.New2Q[string, []byte](capacity) //nolint:errcheck // capacity always valid
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Add(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runS4LRU(capacity, valSize int) (int, any) {
	// s4lru divides capacity across 4 segments, so multiply by 4
	c := s4lru.New(capacity * 4)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runClock(capacity, valSize int) (int, any) {
	c := clock.NewCache[string, []byte](clock.WithCapacity(capacity))
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runLRU(capacity, valSize int) (int, any) {
	c, _ := lru2.New[string, []byte](capacity) //nolint:errcheck // capacity always valid
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Add(key, val)
	}
	return c.Len(), c
}

//nolint:gocritic // unnamed results and eval order are intentional
func runTTLCache(capacity, valSize int) (int, any) {
	c := ttlcache.New[string, []byte](
		ttlcache.WithCapacity[string, []byte](uint64(capacity)),
		ttlcache.WithTTL[string, []byte](time.Hour),
	)
	go c.Start()

	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		val := make([]byte, valSize)
		for j := range val {
			val[j] = byte(i + j)
		}
		c.Set(key, val, ttlcache.DefaultTTL)
	}
	return c.Len(), c
}
