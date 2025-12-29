package trace

import (
	_ "embed"
	"sync"
)

//go:embed testdata/wiki_trace.csv.zst
var wikipediaTraceCompressed []byte

var (
	wikipediaTraceOps  []string
	wikipediaTraceOnce sync.Once
	errWikipediaTrace  error
)

// WikipediaInfo returns information about the Wikipedia trace.
func WikipediaInfo() string {
	return "Wikipedia CDN trace (4M requests, ~1.7M unique paths)"
}

// LoadWikipediaTrace decompresses and parses the embedded Wikipedia trace data.
func LoadWikipediaTrace() ([]string, error) {
	wikipediaTraceOnce.Do(func() {
		wikipediaTraceOps, errWikipediaTrace = loadSimpleTrace(wikipediaTraceCompressed, 4_000_000)
	})
	return wikipediaTraceOps, errWikipediaTrace
}
