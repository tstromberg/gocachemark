package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed testdata/thesios_trace.csv.zst
var thesiosTraceCompressed []byte

var (
	thesiosBlockOps  []string
	thesiosFileOps   []string
	thesiosTraceOnce sync.Once
	errThesiosTrace  error
)

// ThesiosBlockInfo returns information about the Thesios block trace.
func ThesiosBlockInfo() string {
	return "Google Thesios I/O block trace (1.7M reads, ~1.3M unique blocks)"
}

// ThesiosFileInfo returns information about the Thesios file trace.
func ThesiosFileInfo() string {
	return "Google Thesios I/O file trace (1.4M reads, ~131K unique files, sequential deduped)"
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
		blockOps := make([]string, 0, 1_800_000)
		fileOps := make([]string, 0, 1_400_000)

		var lastFile string
		var lastOffset int64 = -1

		for scanner.Scan() {
			key := scanner.Text()
			if key == "" {
				continue
			}

			blockOps = append(blockOps, key)

			// Parse filename and offset for sequential deduping
			idx := strings.LastIndex(key, ":")
			if idx <= 0 {
				fileOps = append(fileOps, key)
				lastFile = key
				lastOffset = -1
				continue
			}

			filename := key[:idx]
			offset, err := strconv.ParseInt(key[idx+1:], 10, 64)
			if err != nil {
				fileOps = append(fileOps, filename)
				lastFile = filename
				lastOffset = -1
				continue
			}

			// Skip if same file with increasing offset (sequential read)
			if filename == lastFile && offset > lastOffset {
				lastOffset = offset
				continue
			}

			fileOps = append(fileOps, filename)
			lastFile = filename
			lastOffset = offset
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
