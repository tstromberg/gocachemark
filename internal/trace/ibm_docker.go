package trace

import (
	_ "embed"
	"sync"
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
		ibmDockerTraceOps, errIBMDockerTrace = loadSimpleTrace(ibmDockerTraceCompressed, 725_000)
	})
	return ibmDockerTraceOps, errIBMDockerTrace
}
