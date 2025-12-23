package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed testdata/tencent_photo_2m.txt.zst
var tencentPhotoTraceCompressed []byte

var (
	tencentPhotoTraceOps  []string
	tencentPhotoTraceOnce sync.Once
	errTencentPhotoTrace  error
)

// TencentPhotoInfo returns information about the Tencent Photo trace.
func TencentPhotoInfo() string {
	return "Tencent Photo trace (2M requests, ~1.34M unique photos)"
}

// LoadTencentPhotoTrace decompresses and parses the embedded Tencent Photo trace data.
func LoadTencentPhotoTrace() ([]string, error) {
	tencentPhotoTraceOnce.Do(func() {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errTencentPhotoTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(tencentPhotoTraceCompressed, nil)
		if err != nil {
			errTencentPhotoTrace = fmt.Errorf("decompress trace: %w", err)
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
			errTencentPhotoTrace = fmt.Errorf("scan trace: %w", err)
			return
		}

		tencentPhotoTraceOps = ops
	})

	return tencentPhotoTraceOps, errTencentPhotoTrace
}
