package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
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
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errTwitterTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(twitterTraceCompressed, nil)
		if err != nil {
			errTwitterTrace = fmt.Errorf("decompress trace: %w", err)
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
			errTwitterTrace = fmt.Errorf("scan trace: %w", err)
			return
		}

		twitterTraceOps = ops
	})

	return twitterTraceOps, errTwitterTrace
}
