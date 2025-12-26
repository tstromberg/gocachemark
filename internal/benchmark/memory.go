package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"

	"github.com/tstromberg/gocachemark/internal/cache"
)

// MemoryResult holds memory usage results for a cache.
type MemoryResult struct {
	Name          string `json:"name"`
	Items         int    `json:"items"`
	Bytes         uint64 `json:"bytes"`
	BytesPerItem  int64  `json:"bytesPerItem"`
	BaselineBytes uint64 `json:"baselineBytes"`
}

type memOutput struct {
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
	Items int    `json:"items"`
	Bytes uint64 `json:"bytes"`
}

// DefaultMemoryCapacity is the cache size for memory benchmarks.
const DefaultMemoryCapacity = 32768

// DefaultValueSize is the value size in bytes.
const DefaultValueSize = 1024

// RunMemory benchmarks memory usage for all caches using isolated processes.
func RunMemory(capacity, valSize int) ([]MemoryResult, error) {
	// Run go mod tidy first to ensure dependencies are resolved
	tidyCmd := exec.Command("go", "mod", "tidy") //nolint:noctx // trusted command
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy: %w\n%s", err, out)
	}

	// Build the memory benchmark binary
	binPath := "./mem-benchmark"
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/mem") //nolint:noctx // trusted command
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build mem benchmark: %w\n%s", err, out)
	}
	defer os.Remove(binPath) //nolint:errcheck // best-effort cleanup

	names := cache.AllNames()
	results := make([]MemoryResult, 0, len(names))

	for _, name := range names {
		res, err := runMemBenchmark(binPath, name, capacity, valSize)
		if err != nil {
			fmt.Printf("  %s: error: %v\n", name, err)
			continue
		}
		results = append(results, res)
	}

	// Get baseline for overhead calculation
	baseline, err := runMemBenchmark(binPath, "baseline", capacity, valSize)
	if err != nil {
		return nil, fmt.Errorf("baseline benchmark: %w", err)
	}

	// Calculate overhead per item
	for i := range results {
		results[i].BaselineBytes = baseline.Bytes
		if results[i].Items > 0 {
			diff := int64(results[i].Bytes) - int64(baseline.Bytes) //nolint:gosec // safe conversion
			results[i].BytesPerItem = diff / int64(results[i].Items)
		}
	}

	// Sort by bytes ascending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Bytes < results[j].Bytes
	})

	return results, nil
}

func runMemBenchmark(binPath, cacheName string, capacity, valSize int) (MemoryResult, error) {
	cmd := exec.Command(binPath, //nolint:gosec,noctx // trusted binary path
		"-cache", cacheName,
		"-cap", strconv.Itoa(capacity),
		"-valSize", strconv.Itoa(valSize),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return MemoryResult{}, fmt.Errorf("run %s: %w\n%s", cacheName, err, out)
	}

	var res memOutput
	if err := json.Unmarshal(out, &res); err != nil {
		return MemoryResult{}, fmt.Errorf("parse output for %s: %w\n%s", cacheName, err, out)
	}

	if res.Error != "" {
		return MemoryResult{}, fmt.Errorf("%s: %s", cacheName, res.Error)
	}

	return MemoryResult{
		Name:  res.Name,
		Items: res.Items,
		Bytes: res.Bytes,
	}, nil
}
