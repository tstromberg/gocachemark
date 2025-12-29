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

//go:embed testdata/thesios_trace_part1.csv.zst
var thesiosTracePart1 []byte

//go:embed testdata/thesios_trace_part2.csv.zst
var thesiosTracePart2 []byte

var (
	thesiosBlockOps  []string
	thesiosFileOps   []string
	thesiosTraceOnce sync.Once
	errThesiosTrace  error
)

// ThesiosBlockInfo returns information about the Thesios block trace.
func ThesiosBlockInfo() string {
	return "Google Thesios I/O block trace (15.3M reads, ~10.2M unique blocks)"
}

// ThesiosFileInfo returns information about the Thesios file trace.
func ThesiosFileInfo() string {
	return "Google Thesios I/O file trace (15.3M reads, ~603K unique files, sequential deduped)"
}

func loadThesiosTrace() {
	thesiosTraceOnce.Do(func() {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errThesiosTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		part1, err := decoder.DecodeAll(thesiosTracePart1, nil)
		if err != nil {
			errThesiosTrace = fmt.Errorf("decompress part 1: %w", err)
			return
		}

		part2, err := decoder.DecodeAll(thesiosTracePart2, nil)
		if err != nil {
			errThesiosTrace = fmt.Errorf("decompress part 2: %w", err)
			return
		}

		decompressed := make([]byte, len(part1)+len(part2))
		copy(decompressed, part1)
		copy(decompressed[len(part1):], part2)

		scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
		blockOps := make([]string, 0, 16_000_000)
		fileOps := make([]string, 0, 12_000_000)

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
