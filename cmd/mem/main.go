// Package main measures memory usage for a single cache implementation.
// Run in isolated process for accurate measurements.
package main

import (
	"flag"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/Code-Hex/go-generics-cache/policy/clock"
	"github.com/Yiling-J/theine-go"
	"github.com/codeGROOVE-dev/sfcache"
	"github.com/coocood/freecache"
	"github.com/dgraph-io/ristretto"
	"github.com/dgryski/go-s4lru"
	"github.com/elastic/go-freelru"
	lru2 "github.com/hashicorp/golang-lru/v2"
	"github.com/maypok86/otter/v2"
	"github.com/scalalang2/golang-fifo/s3fifo"
	"github.com/scalalang2/golang-fifo/sieve"
	tinylfu "github.com/vmihailenco/go-tinylfu"
	"github.com/zeebo/xxh3"
)

var keepAlive any

func main() {
	cacheName := flag.String("cache", "", "cache implementation to benchmark")
	capacity := flag.Int("cap", 32768, "capacity")
	valSize := flag.Int("valSize", 1024, "value size in bytes")
	flag.Parse()

	if *cacheName == "" {
		fmt.Println(`{"error":"cache name required"}`)
		return
	}

	runtime.GC()
	debug.FreeOSMemory()

	items := runBenchmark(*cacheName, *capacity, *valSize)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	debug.FreeOSMemory()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	fmt.Printf(`{"name":%q, "items":%d, "bytes":%d}`, *cacheName, items, mem.Alloc)
}

func runBenchmark(name string, capacity, valSize int) int {
	switch name {
	case "baseline":
		return runBaseline(capacity, valSize)
	case "sfcache":
		return runSFCache(capacity, valSize)
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
	default:
		return 0
	}
}

func runBaseline(capacity, valSize int) int {
	m := make(map[string][]byte, capacity)
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		m[key] = make([]byte, valSize)
	}
	keepAlive = m
	return len(m)
}

func runSFCache(capacity, valSize int) int {
	c := sfcache.New[string, []byte](sfcache.Size(capacity))
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize))
			} else {
				c.Set(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}

func runOtter(capacity, valSize int) int {
	c := otter.Must(&otter.Options[string, []byte]{MaximumSize: capacity})
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize))
			} else {
				c.Set(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.EstimatedSize()
}

func runTheine(capacity, valSize int) int {
	c, _ := theine.NewBuilder[string, []byte](int64(capacity)).Build()
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize), 0)
			} else {
				c.Set(key, nil, 0)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize), 0)
	}
	keepAlive = c
	return c.Len()
}

func runRistretto(capacity, valSize int) int {
	c, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(capacity * 10),
		MaxCost:     int64(capacity),
		BufferItems: 64,
	})
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize), 1)
			} else {
				c.Set(key, nil, 1)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize), 1)
	}
	time.Sleep(10 * time.Millisecond)
	keepAlive = c
	return int(c.Metrics.KeysAdded() - c.Metrics.KeysEvicted())
}

func runTinyLFU(capacity, valSize int) int {
	c := tinylfu.NewSync(capacity, capacity*10)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(&tinylfu.Item{Key: key, Value: make([]byte, valSize)})
			} else {
				c.Set(&tinylfu.Item{Key: key, Value: nil})
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(&tinylfu.Item{Key: key, Value: make([]byte, valSize)})
	}
	keepAlive = c
	// tinylfu.SyncT doesn't expose Len, return capacity
	return capacity
}

func runSieve(capacity, valSize int) int {
	c := sieve.New[string, []byte](capacity, 0)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize))
			} else {
				c.Set(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}

func runS3FIFO(capacity, valSize int) int {
	c := s3fifo.New[string, []byte](capacity, 0)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize))
			} else {
				c.Set(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}

func hashString(s string) uint32 {
	return uint32(xxh3.HashString(s))
}

func runFreeLRUSharded(capacity, valSize int) int {
	c, _ := freelru.NewSharded[string, []byte](uint32(capacity), hashString)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Add(key, make([]byte, valSize))
			} else {
				c.Add(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Add(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}

func runFreeLRUSynced(capacity, valSize int) int {
	c, _ := freelru.NewSynced[string, []byte](uint32(capacity), hashString)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Add(key, make([]byte, valSize))
			} else {
				c.Add(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Add(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}

func runFreecache(capacity, valSize int) int {
	overhead := 256
	size := capacity * (valSize + overhead)
	c := freecache.NewCache(size)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set([]byte(key), make([]byte, valSize), 0)
			} else {
				c.Set([]byte(key), nil, 0)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set([]byte(key), make([]byte, valSize), 0)
	}
	keepAlive = c
	return int(c.EntryCount())
}

func runTwoQueue(capacity, valSize int) int {
	c, _ := lru2.New2Q[string, []byte](capacity)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Add(key, make([]byte, valSize))
			} else {
				c.Add(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Add(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}

func runS4LRU(capacity, valSize int) int {
	c := s4lru.New(capacity)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Set(key, make([]byte, valSize))
			} else {
				c.Set(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Set(key, make([]byte, valSize))
	}
	keepAlive = c
	return capacity
}

func runClock(capacity, valSize int) int {
	c := clock.NewCache[string, []byte](clock.WithCapacity(capacity))
	var mu sync.Mutex
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			mu.Lock()
			if pass == 0 {
				c.Set(key, make([]byte, valSize))
			} else {
				c.Set(key, nil)
			}
			mu.Unlock()
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		mu.Lock()
		c.Set(key, make([]byte, valSize))
		mu.Unlock()
	}
	keepAlive = c
	return capacity
}

func runLRU(capacity, valSize int) int {
	c, _ := lru2.New[string, []byte](capacity)
	for pass := range 3 {
		for i := range capacity {
			key := "key-" + strconv.Itoa(i)
			if pass == 0 {
				c.Add(key, make([]byte, valSize))
			} else {
				c.Add(key, nil)
			}
		}
	}
	for i := range capacity {
		key := "key-" + strconv.Itoa(i)
		c.Add(key, make([]byte, valSize))
	}
	keepAlive = c
	return c.Len()
}
