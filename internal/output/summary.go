package output

import (
	"sort"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

// CategorySummary represents a cache's average performance across a category.
type CategorySummary struct {
	Name          string  // cache name
	Value         float64 // average value (hit rate %, latency ns, throughput QPS)
	DiffFromFirst float64 // relative % difference from first place (0 for first)
}

// ComputeHitRateSummary computes average hit rate across all traces for each cache.
// Returns (summaries, numTests, filtered) where filtered is true if some caches were excluded.
func ComputeHitRateSummary(data *HitRateData) ([]CategorySummary, int, bool) {
	if data == nil {
		return nil, 0, false
	}

	// Collect all traces that have results
	traces := [][]benchmark.HitRateResult{
		data.CDN, data.Meta, data.Zipf, data.Twitter, data.Wikipedia,
		data.ThesiosBlock, data.ThesiosFile, data.IBMDocker, data.TencentPhoto,
	}

	// Count how many tests were run
	n := 0
	for _, tr := range traces {
		if len(tr) > 0 {
			n++
		}
	}
	if n == 0 {
		return nil, 0, false
	}

	// Count results per cache
	sums := make(map[string]float64)
	counts := make(map[string]int)

	for _, tr := range traces {
		for _, r := range tr {
			avg := AvgHitRate(r, data.Sizes)
			sums[r.Name] += avg
			counts[r.Name]++
		}
	}

	// Only include caches that participated in ALL tests
	var out []CategorySummary
	filtered := false
	for name, sum := range sums {
		cnt := counts[name]
		if cnt < n {
			filtered = true
			continue
		}
		out = append(out, CategorySummary{
			Name:  name,
			Value: sum / float64(cnt),
		})
	}

	if len(out) < 2 {
		return nil, 0, false
	}

	// Sort by hit rate descending (higher is better)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Value > out[j].Value
	})

	// Calculate absolute difference from first place (percentage points)
	best := out[0].Value
	for i := range out {
		out[i].DiffFromFirst = best - out[i].Value
	}

	return out, n, filtered
}

// ComputeLatencySummary computes average latency across all tested latency benchmarks.
// Only includes caches that participated in all tests.
// Returns (summaries, numTests, filtered) where filtered is true if some caches were excluded.
func ComputeLatencySummary(data *LatencyData) ([]CategorySummary, int, bool) {
	if data == nil {
		return nil, 0, false
	}

	// Count how many test types were run
	n := 0
	if len(data.Results) > 0 {
		n++
	}
	if len(data.IntResults) > 0 {
		n++
	}
	if len(data.GetOrSetResults) > 0 {
		n++
	}
	if n == 0 {
		return nil, 0, false
	}

	// Collect results from all test types
	sums := make(map[string]float64)
	counts := make(map[string]int)

	for _, r := range data.Results {
		avg := (r.GetNsOp + r.SetNsOp) / 2
		sums[r.Name] += avg
		counts[r.Name]++
	}
	for _, r := range data.IntResults {
		avg := (r.GetNsOp + r.SetNsOp) / 2
		sums[r.Name] += avg
		counts[r.Name]++
	}
	for _, r := range data.GetOrSetResults {
		sums[r.Name] += r.NsOp
		counts[r.Name]++
	}

	// Only include caches that participated in ALL tests
	var out []CategorySummary
	filtered := false
	for name, sum := range sums {
		cnt := counts[name]
		if cnt < n {
			filtered = true
			continue
		}
		out = append(out, CategorySummary{
			Name:  name,
			Value: sum / float64(cnt),
		})
	}

	if len(out) < 2 {
		return nil, 0, false
	}

	// Sort by latency ascending (lower is better)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Value < out[j].Value
	})

	// Calculate relative difference from first place
	best := out[0].Value
	for i := range out {
		if best > 0 {
			out[i].DiffFromFirst = (out[i].Value - best) / best * 100
		}
	}

	return out, n, filtered
}

// ComputeThroughputSummary computes average throughput across all tested benchmarks.
// Only includes caches that participated in all tests.
// Returns (summaries, numTests, filtered) where filtered is true if some caches were excluded.
func ComputeThroughputSummary(data *ThroughputData) ([]CategorySummary, int, bool) {
	if data == nil {
		return nil, 0, false
	}

	// Count how many test types were run
	tests := 0
	if len(data.StringGetResults) > 0 {
		tests++
	}
	if len(data.StringSetResults) > 0 {
		tests++
	}
	if len(data.IntGetResults) > 0 {
		tests++
	}
	if len(data.IntSetResults) > 0 {
		tests++
	}
	if len(data.GetOrSetResults) > 0 {
		tests++
	}
	if tests == 0 {
		return nil, 0, false
	}

	// Collect results from all test types
	sums := make(map[string]float64)
	counts := make(map[string]int)

	for _, r := range data.StringGetResults {
		sums[r.Name] += avgQPS(r)
		counts[r.Name]++
	}
	for _, r := range data.StringSetResults {
		sums[r.Name] += avgQPS(r)
		counts[r.Name]++
	}
	for _, r := range data.IntGetResults {
		sums[r.Name] += avgQPS(r)
		counts[r.Name]++
	}
	for _, r := range data.IntSetResults {
		sums[r.Name] += avgQPS(r)
		counts[r.Name]++
	}
	for _, r := range data.GetOrSetResults {
		sums[r.Name] += avgQPS(r)
		counts[r.Name]++
	}

	// Only include caches that participated in ALL tests
	var out []CategorySummary
	filtered := false
	for name, sum := range sums {
		cnt := counts[name]
		if cnt < tests {
			filtered = true
			continue
		}
		out = append(out, CategorySummary{
			Name:  name,
			Value: sum / float64(cnt),
		})
	}

	if len(out) < 2 {
		return nil, 0, false
	}

	// Sort by QPS descending (higher is better)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Value > out[j].Value
	})

	// Calculate relative difference from first place
	best := out[0].Value
	for i := range out {
		if best > 0 {
			out[i].DiffFromFirst = (best - out[i].Value) / best * 100
		}
	}

	return out, tests, filtered
}
