package trace

import (
	_ "embed"
	"sync"
)

//go:embed testdata/tencent_photo_trace_part1.csv.zst
var tencentPhotoTracePart1 []byte

//go:embed testdata/tencent_photo_trace_part2.csv.zst
var tencentPhotoTracePart2 []byte

//go:embed testdata/tencent_photo_trace_part3.csv.zst
var tencentPhotoTracePart3 []byte

var (
	tencentPhotoTraceOps  []string
	tencentPhotoTraceOnce sync.Once
	errTencentPhotoTrace  error
)

// TencentPhotoInfo returns information about the Tencent Photo trace.
func TencentPhotoInfo() string {
	return "Tencent Photo trace (8M requests, ~4.4M unique photos)"
}

// LoadTencentPhotoTrace decompresses and parses the embedded Tencent Photo trace data.
func LoadTencentPhotoTrace() ([]string, error) {
	tencentPhotoTraceOnce.Do(func() {
		tencentPhotoTraceOps, errTencentPhotoTrace = loadMultiPartTrace(
			8_000_000,
			tencentPhotoTracePart1,
			tencentPhotoTracePart2,
			tencentPhotoTracePart3,
		)
	})
	return tencentPhotoTraceOps, errTencentPhotoTrace
}
