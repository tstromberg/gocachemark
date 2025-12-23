package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed testdata/thesios_trace_400k.csv.zst
var thesiosTraceCompressed []byte

var (
	thesiosBlockOps  []string
	thesiosFileOps   []string
	thesiosTraceOnce sync.Once
	errThesiosTrace  error
)

// ThesiosBlockInfo returns information about the Thesios block trace.
func ThesiosBlockInfo() string {
	return "Google Thesios I/O block trace (400K reads, ~322K unique blocks)"
}

// ThesiosFileInfo returns information about the Thesios file trace.
func ThesiosFileInfo() string {
	return "Google Thesios I/O file trace (400K reads, ~46K unique files)"
}

func loadThesiosTrace() {
	thesiosTraceOnce.Do(func() {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errThesiosTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(thesiosTraceCompressed, nil)
		if err != nil {
			errThesiosTrace = fmt.Errorf("decompress trace: %w", err)
			return
		}

		scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
		blockOps := make([]string, 0, 400_000)
		fileOps := make([]string, 0, 400_000)

		for scanner.Scan() {
			key := scanner.Text()
			if key != "" {
				blockOps = append(blockOps, key)
				// Strip offset to get filename only
				if idx := strings.LastIndex(key, ":"); idx > 0 {
					fileOps = append(fileOps, key[:idx])
				} else {
					fileOps = append(fileOps, key)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errThesiosTrace = fmt.Errorf("scan trace: %w", err)
			return
		}

		thesiosBlockOps = blockOps
		thesiosFileOps = fileOps
	})
}

// LoadThesiosBlockTrace returns the block-level trace (filename:offset keys).
func LoadThesiosBlockTrace() ([]string, error) {
	loadThesiosTrace()
	return thesiosBlockOps, errThesiosTrace
}

// LoadThesiosFileTrace returns the file-level trace (filename-only keys).
func LoadThesiosFileTrace() ([]string, error) {
	loadThesiosTrace()
	return thesiosFileOps, errThesiosTrace
}
