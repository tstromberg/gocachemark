package trace

import (
	_ "embed"
	"sync"
)

//go:embed testdata/twitter_trace_2m.csv.zst
var twitterTraceCompressed []byte

var (
	twitterTraceOps  []string
	twitterTraceOnce sync.Once
	errTwitterTrace  error
)

// TwitterInfo returns information about the Twitter trace.
func TwitterInfo() string {
	return "Twitter production cache trace (2M ops, cluster001+cluster052)"
}

// LoadTwitterTrace decompresses and parses the embedded Twitter trace data.
func LoadTwitterTrace() ([]string, error) {
	twitterTraceOnce.Do(func() {
		twitterTraceOps, errTwitterTrace = loadSimpleTrace(twitterTraceCompressed, 2_000_000)
	})
	return twitterTraceOps, errTwitterTrace
}
