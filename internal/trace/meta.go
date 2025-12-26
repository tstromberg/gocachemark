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

// Op represents a single cache operation from a trace.
type Op struct {
	Key    string
	Action string // "GET" or "SET"
}

var (
	metaTraceOps  []Op
	metaTraceOnce sync.Once
	errMetaTrace  error
)

// MetaInfo returns information about the Meta trace.
func MetaInfo() string {
	return "Meta KVCache production trace (3M ops)"
}

// LoadMetaTrace decompresses and parses the embedded Meta trace data.
func LoadMetaTrace() ([]Op, error) {
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
		ops := make([]Op, 0, 3_000_000)

		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, ",", 2)
			if len(parts) < 2 {
				continue
			}
			ops = append(ops, Op{
				Key:    parts[0],
				Action: parts[1],
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
