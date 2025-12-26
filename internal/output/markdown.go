package output

import (
	"fmt"
	"os"
	"sort"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

// WriteMarkdown writes benchmark results to a Markdown file.
func WriteMarkdown(filename string, results Results, commandLine string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := func(format string, args ...any) {
		fmt.Fprintf(f, format, args...)
	}

	w("# gocachemark Results\n\n")
	w("```\n")
	w("Command: %s\n", commandLine)
	w("Environment: %s/%s, %d CPUs, %s\n", results.MachineInfo.OS, results.MachineInfo.Arch, results.MachineInfo.NumCPU, results.MachineInfo.GoVersion)
	w("```\n\n")

	// Hit Rate
	if results.HitRate != nil {
		w("## Hit Rate Benchmarks\n\n")
		writeHitRateMarkdown(w, "CDN", results.HitRate.CDN, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Meta", results.HitRate.Meta, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Zipf", results.HitRate.Zipf, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Twitter", results.HitRate.Twitter, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Wikipedia", results.HitRate.Wikipedia, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Thesios Block", results.HitRate.ThesiosBlock, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Thesios File", results.HitRate.ThesiosFile, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "IBM Docker", results.HitRate.IBMDocker, results.HitRate.Sizes)
		writeHitRateMarkdown(w, "Tencent Photo", results.HitRate.TencentPhoto, results.HitRate.Sizes)
	}

	// Latency
	if results.Latency != nil {
		w("## Latency Benchmarks\n\n")
		writeLatencyMarkdown(w, results.Latency.Results)
		writeIntLatencyMarkdown(w, results.Latency.IntResults)
		writeGetOrSetLatencyMarkdown(w, results.Latency.GetOrSetResults)
	}

	// Throughput
	if results.Throughput != nil {
		w("## Throughput Benchmarks\n\n")
		writeThroughputMarkdown(w, "String Get", results.Throughput.StringGetResults, results.Throughput.Threads)
		writeThroughputMarkdown(w, "String Set", results.Throughput.StringSetResults, results.Throughput.Threads)
		writeThroughputMarkdown(w, "Int Get", results.Throughput.IntGetResults, results.Throughput.Threads)
		writeThroughputMarkdown(w, "Int Set", results.Throughput.IntSetResults, results.Throughput.Threads)
		writeThroughputMarkdown(w, "GetOrSet", results.Throughput.GetOrSetResults, results.Throughput.Threads)
	}

	// Memory
	if results.Memory != nil && len(results.Memory.Results) > 0 {
		w("## Memory Benchmarks\n\n")
		writeMemoryMarkdown(w, results.Memory.Results)
	}

	// Rankings
	if len(results.Rankings) > 0 {
		w("## Overall Rankings\n\n")
		w("| Rank | Cache         | Score | Gold | Silver | Bronze |\n")
		w("|------|---------------|-------|------|--------|--------|\n")
		for _, r := range results.Rankings {
			w("| %4d | %-13s | %5.0f | %4d | %6d | %6d |\n", r.Rank, r.Name, r.Score, r.Gold, r.Silver, r.Bronze)
		}
		w("\n")
	}

	return nil
}

func writeHitRateMarkdown(w func(string, ...any), name string, data []benchmark.HitRateResult, sizes []int) {
	if len(data) == 0 {
		return
	}

	w("### %s\n\n", name)

	// Header
	w("| Cache         |")
	for _, size := range sizes {
		w(" %5dK |", size/1024)
	}
	w("    Avg |\n")

	// Separator
	w("|---------------|")
	for range sizes {
		w("--------|")
	}
	w("--------|\n")

	// Sort by average
	sorted := make([]benchmark.HitRateResult, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool {
		return avgHitRate(sorted[i], sizes) > avgHitRate(sorted[j], sizes)
	})

	// Data rows
	for _, r := range sorted {
		w("| %-13s |", r.Name)
		for _, size := range sizes {
			w(" %5.2f%% |", r.Rates[size])
		}
		w(" %5.2f%% |\n", avgHitRate(r, sizes))
	}

	// Winner line
	if len(sorted) >= 2 {
		best, second := sorted[0], sorted[1]
		bestAvg := avgHitRate(best, sizes)
		secondAvg := avgHitRate(second, sizes)
		pct := ((bestAvg - secondAvg) / secondAvg) * 100
		w("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
	}
	w("\n")
}

func writeLatencyMarkdown(w func(string, ...any), data []benchmark.LatencyResult) {
	if len(data) == 0 {
		return
	}

	w("### String Keys\n\n")
	w("| Cache         | Get ns | Get alloc | Set ns | Set alloc | SetEvict ns | SetEvict alloc | Avg ns |\n")
	w("|---------------|--------|-----------|--------|-----------|-------------|----------------|--------|\n")

	sorted := make([]benchmark.LatencyResult, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool {
		return (sorted[i].GetNsOp + sorted[i].SetNsOp) < (sorted[j].GetNsOp + sorted[j].SetNsOp)
	})

	for _, r := range sorted {
		avg := (r.GetNsOp + r.SetNsOp) / 2
		w("| %-13s | %6.0f | %9d | %6.0f | %9d | %11.0f | %14d | %6.0f |\n",
			r.Name, r.GetNsOp, r.GetAllocs, r.SetNsOp, r.SetAllocs, r.SetEvictNsOp, r.SetEvictAllocs, avg)
	}

	// Winner line
	if len(sorted) >= 2 {
		best, second := sorted[0], sorted[1]
		bestAvg := (best.GetNsOp + best.SetNsOp) / 2
		secondAvg := (second.GetNsOp + second.SetNsOp) / 2
		pct := ((secondAvg - bestAvg) / bestAvg) * 100
		w("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
	}
	w("\n")
}

func writeIntLatencyMarkdown(w func(string, ...any), data []benchmark.IntLatencyResult) {
	if len(data) == 0 {
		return
	}

	w("### Int Keys\n\n")
	w("| Cache         | Get ns | Get alloc | Set ns | Set alloc | Avg ns |\n")
	w("|---------------|--------|-----------|--------|-----------|--------|\n")

	sorted := make([]benchmark.IntLatencyResult, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool {
		return (sorted[i].GetNsOp + sorted[i].SetNsOp) < (sorted[j].GetNsOp + sorted[j].SetNsOp)
	})

	for _, r := range sorted {
		avg := (r.GetNsOp + r.SetNsOp) / 2
		w("| %-13s | %6.0f | %9d | %6.0f | %9d | %6.0f |\n",
			r.Name, r.GetNsOp, r.GetAllocs, r.SetNsOp, r.SetAllocs, avg)
	}

	// Winner line
	if len(sorted) >= 2 {
		best, second := sorted[0], sorted[1]
		bestAvg := (best.GetNsOp + best.SetNsOp) / 2
		secondAvg := (second.GetNsOp + second.SetNsOp) / 2
		pct := ((secondAvg - bestAvg) / bestAvg) * 100
		w("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
	}
	w("\n")
}

func writeGetOrSetLatencyMarkdown(w func(string, ...any), data []benchmark.GetOrSetLatencyResult) {
	if len(data) == 0 {
		return
	}

	w("### GetOrSet\n\n")
	w("| Cache         | GetOrSet ns | GetOrSet alloc |\n")
	w("|---------------|-------------|----------------|\n")

	sorted := make([]benchmark.GetOrSetLatencyResult, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].NsOp < sorted[j].NsOp
	})

	for _, r := range sorted {
		w("| %-13s | %11.0f | %14d |\n", r.Name, r.NsOp, r.Allocs)
	}

	// Winner line
	if len(sorted) >= 2 {
		best, second := sorted[0], sorted[1]
		pct := ((second.NsOp - best.NsOp) / best.NsOp) * 100
		w("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
	}
	w("\n")
}

func writeThroughputMarkdown(w func(string, ...any), name string, data []benchmark.ThroughputResult, threads []int) {
	if len(data) == 0 {
		return
	}

	w("### %s\n\n", name)

	// Header
	w("| Cache         |")
	for _, t := range threads {
		w(" %2dT       |", t)
	}
	w("       Avg |\n")

	// Separator
	w("|---------------|")
	for range threads {
		w("-----------|")
	}
	w("-----------|\n")

	// Sort by average
	sorted := make([]benchmark.ThroughputResult, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool {
		return avgQPS(sorted[i]) > avgQPS(sorted[j])
	})

	// Data rows
	for _, r := range sorted {
		w("| %-13s |", r.Name)
		for _, t := range threads {
			qps := r.QPS[t]
			if qps >= 1_000_000 {
				w(" %6.2fM   |", qps/1_000_000)
			} else {
				w(" %6.0fK   |", qps/1_000)
			}
		}
		avg := avgQPS(r)
		if avg >= 1_000_000 {
			w(" %6.2fM   |\n", avg/1_000_000)
		} else {
			w(" %6.0fK   |\n", avg/1_000)
		}
	}

	// Winner line
	if len(sorted) >= 2 {
		best, second := sorted[0], sorted[1]
		bestAvg := avgQPS(best)
		secondAvg := avgQPS(second)
		pct := ((bestAvg - secondAvg) / secondAvg) * 100
		w("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
	}
	w("\n")
}

func writeMemoryMarkdown(w func(string, ...any), data []benchmark.MemoryResult) {
	if len(data) == 0 {
		return
	}

	w("| Cache         | Items Stored | Memory (MB) | Overhead (bytes/item) |\n")
	w("|---------------|--------------|-------------|-----------------------|\n")

	for _, r := range data {
		mb := float64(r.Bytes) / 1024 / 1024
		w("| %-13s | %12d | %11.2f | %21d |\n", r.Name, r.Items, mb, r.BytesPerItem)
	}

	// Winner line (lowest memory usage)
	if len(data) >= 2 {
		best, second := data[0], data[1]
		pct := (float64(second.Bytes-best.Bytes) / float64(best.Bytes)) * 100
		w("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
	}
	w("\n")
}

func avgQPS(r benchmark.ThroughputResult) float64 {
	var sum float64
	for _, qps := range r.QPS {
		sum += qps
	}
	return sum / float64(len(r.QPS))
}
