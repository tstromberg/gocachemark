package trace

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

//go:embed testdata/ibm_docker_trace_725k.csv.zst
var ibmDockerTraceCompressed []byte

var (
	ibmDockerTraceOps  []string
	ibmDockerTraceOnce sync.Once
	errIBMDockerTrace  error
)

// IBMDockerInfo returns information about the IBM Docker Registry trace.
func IBMDockerInfo() string {
	return "IBM Docker Registry trace (725K GETs, ~121K unique URIs)"
}

// LoadIBMDockerTrace decompresses and parses the embedded IBM Docker Registry trace data.
func LoadIBMDockerTrace() ([]string, error) {
	ibmDockerTraceOnce.Do(func() {
		decoder, err := zstd.NewReader(nil)
		if err != nil {
			errIBMDockerTrace = fmt.Errorf("create zstd decoder: %w", err)
			return
		}
		defer decoder.Close()

		decompressed, err := decoder.DecodeAll(ibmDockerTraceCompressed, nil)
		if err != nil {
			errIBMDockerTrace = fmt.Errorf("decompress trace: %w", err)
			return
		}

		scanner := bufio.NewScanner(strings.NewReader(string(decompressed)))
		ops := make([]string, 0, 725_000)

		for scanner.Scan() {
			key := scanner.Text()
			if key != "" {
				ops = append(ops, key)
			}
		}

		if err := scanner.Err(); err != nil {
			errIBMDockerTrace = fmt.Errorf("scan trace: %w", err)
			return
		}

		ibmDockerTraceOps = ops
	})

	return ibmDockerTraceOps, errIBMDockerTrace
}
