package trace

import (
	_ "embed"
	"sync"
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
		wikipediaTraceOps, errWikipediaTrace = loadSimpleTrace(wikipediaTraceCompressed, 2_000_000)
	})
	return wikipediaTraceOps, errWikipediaTrace
}
