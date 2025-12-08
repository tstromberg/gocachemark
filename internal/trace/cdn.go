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

//go:embed testdata/reag0c01_20230315_20230322_0.2000.csv.zstd
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
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errCDNTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(cdnTraceCompressed, nil)
		if err != nil {
			errCDNTrace = fmt.Errorf("decompress trace: %w", err)
			return
		}

		scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
		ops := make([]string, 0, 2_000_000)

		// Skip header
		if scanner.Scan() {
			// header line - skip
		}

		for scanner.Scan() {
			line := scanner.Text()
			// Find first and second comma to extract cacheKey
			firstComma := strings.Index(line, ",")
			if firstComma < 0 {
				continue
			}
			rest := line[firstComma+1:]
			secondComma := strings.Index(rest, ",")
			if secondComma < 0 {
				continue
			}
			key := rest[:secondComma]
			if key != "" {
				ops = append(ops, key)
			}
		}

		if err := scanner.Err(); err != nil {
			errCDNTrace = fmt.Errorf("scan trace: %w", err)
			return
		}

		cdnTraceOps = ops
	})

	return cdnTraceOps, errCDNTrace
}
