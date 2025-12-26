package trace

import (
	_ "embed"
	"sync"
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
		tencentPhotoTraceOps, errTencentPhotoTrace = loadSimpleTrace(tencentPhotoTraceCompressed, 2_000_000)
	})
	return tencentPhotoTraceOps, errTencentPhotoTrace
}
