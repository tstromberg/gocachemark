package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed testdata/meta_trace_3m.csv.zst
var metaTraceCompressed []byte

// TraceOp represents a single cache operation from a trace.
type TraceOp struct {
	Key string
	Op  string // "GET" or "SET"
}

var (
	metaTraceOps  []TraceOp
	metaTraceOnce sync.Once
	errMetaTrace  error
)

// MetaInfo returns information about the Meta trace.
func MetaInfo() string {
	return "Meta KVCache production trace (3M ops)"
}

// LoadMetaTrace decompresses and parses the embedded Meta trace data.
func LoadMetaTrace() ([]TraceOp, error) {
	metaTraceOnce.Do(func() {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errMetaTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(metaTraceCompressed, nil)
		if err != nil {
			errMetaTrace = fmt.Errorf("decompress trace: %w", err)
			return
		}

		scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
		ops := make([]TraceOp, 0, 3_000_000)

		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, ",", 2)
			if len(parts) < 2 {
				continue
			}
			ops = append(ops, TraceOp{
				Key: parts[0],
				Op:  parts[1],
			})
		}

		if err := scanner.Err(); err != nil {
			errMetaTrace = fmt.Errorf("scan trace: %w", err)
			return
		}
		metaTraceOps = ops
	})

	return metaTraceOps, errMetaTrace
}
