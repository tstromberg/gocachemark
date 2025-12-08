package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed testdata/wiki_trace_2m.csv.zst
var wikipediaTraceCompressed []byte

var (
	wikipediaTraceOps  []string
	wikipediaTraceOnce sync.Once
	errWikipediaTrace  error
)

// WikipediaInfo returns information about the Wikipedia trace.
func WikipediaInfo() string {
	return "Wikipedia CDN upload trace (2M ops, upload.wikimedia.org)"
}

// LoadWikipediaTrace decompresses and parses the embedded Wikipedia trace data.
func LoadWikipediaTrace() ([]string, error) {
	wikipediaTraceOnce.Do(func() {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errWikipediaTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(wikipediaTraceCompressed, nil)
		if err != nil {
			errWikipediaTrace = fmt.Errorf("decompress trace: %w", err)
			return
		}

		scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
		ops := make([]string, 0, 2_000_000)

		for scanner.Scan() {
			key := scanner.Text()
			if key != "" {
				ops = append(ops, key)
			}
		}

		if err := scanner.Err(); err != nil {
			errWikipediaTrace = fmt.Errorf("scan trace: %w", err)
			return
		}

		wikipediaTraceOps = ops
	})

	return wikipediaTraceOps, errWikipediaTrace
}
