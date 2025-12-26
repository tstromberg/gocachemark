// Package trace provides cache trace loading and benchmarking.
package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// loadSimpleTrace decompresses zstd data and parses lines as string keys.
func loadSimpleTrace(compressed []byte, capacity int) ([]string, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("create zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("decompress trace: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
	ops := make([]string, 0, capacity)

	for scanner.Scan() {
		if key := scanner.Text(); key != "" {
			ops = append(ops, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan trace: %w", err)
	}

	return ops, nil
}

//go:embed testdata/reag0c01_keys_only.csv.zst
var cdnTraceCompressed []byte

var (
	cdnTraceOps  []string
	cdnTraceOnce sync.Once
	errCDNTrace  error
)

// CDNInfo returns information about the CDN trace.
func CDNInfo() string {
	return "CDN production trace (2M ops, ~768K unique keys)"
}

// LoadCDNTrace decompresses and parses the embedded CDN trace data.
func LoadCDNTrace() ([]string, error) {
	cdnTraceOnce.Do(func() {
		cdnTraceOps, errCDNTrace = loadSimpleTrace(cdnTraceCompressed, 2_000_000)
	})
	return cdnTraceOps, errCDNTrace
}
