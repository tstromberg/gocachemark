package output

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

// WriteMarkdown writes benchmark results to a Markdown file.
func WriteMarkdown(filename string, results Results, commandLine string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // best-effort close

	w := func(format string, args ...any) {
		fmt.Fprintf(f, format, args...) //nolint:errcheck // best-effort write
	}

	w("# gocachemark Results\n\n")
	w("```\n")
	w("Command: %s\n", commandLine)
	mi := results.MachineInfo
	w("Environment: %s/%s, %d CPUs, %s\n", mi.OS, mi.Arch, mi.NumCPU, mi.GoVersion)
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
		writeLatencyMarkdown(w, "String Keys", results.Latency.Results)
		writeLatencyMarkdown(w, "Int Keys", results.Latency.IntResults)
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
	w("     Avg |\n")

	// Separator
	w("|---------------|")
	for range sizes {
		w("--------|")
	}
	w("---------|\n")

	// Sort by average
	sorted := make([]benchmark.HitRateResult, len(data))
	copy(sorted, data)
	sort.Slice(sorted, func(i, j int) bool {
		return AvgHitRate(sorted[i], sizes) > AvgHitRate(sorted[j], sizes)
	})

	// Data rows
	for _, r := range sorted {
		w("| %-13s |", r.Name)
		for _, size := range sizes {
			w(" %5.2f%% |", r.Rates[size])
		}
		w(" %6.3f%% |\n", AvgHitRate(r, sizes))
	}

	// Winner line
	if len(sorted) >= 1 {
		entries := make([]WinnerEntry, len(sorted))
		for i, r := range sorted {
			entries[i] = WinnerEntry{Name: r.Name, Score: AvgHitRate(r, sizes)}
		}
		winners, runnerUp := FormatWinners(entries)

		if len(winners) > 1 {
			w("\n  winners (tie): %s", strings.Join(winners, ", "))
		} else {
			w("\n  winner: %s", winners[0])
		}
		if runnerUp != nil {
			pct := (entries[0].Score - runnerUp.Score) / runnerUp.Score * 100
			w(" (+%.3f%% vs %s)", pct, runnerUp.Name)
		}
		w("\n")
	}
	w("\n")
}

func writeLatencyMarkdown(w func(string, ...any), title string, data []benchmark.LatencyResult) {
	if len(data) == 0 {
		return
	}

	w("### %s\n\n", title)
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

	if len(sorted) >= 1 {
		entries := make([]WinnerEntry, len(sorted))
		for i, r := range sorted {
			entries[i] = WinnerEntry{Name: r.Name, Score: (r.GetNsOp + r.SetNsOp) / 2}
		}
		winners, runnerUp := FormatWinners(entries)

		if len(winners) > 1 {
			w("\n  winners (tie): %s", strings.Join(winners, ", "))
		} else {
			w("\n  winner: %s", winners[0])
		}
		if runnerUp != nil {
			pct := (runnerUp.Score - entries[0].Score) / entries[0].Score * 100
			w(" (+%.3f%% vs %s)", pct, runnerUp.Name)
		}
		w("\n")
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
	if len(sorted) >= 1 {
		entries := make([]WinnerEntry, len(sorted))
		for i, r := range sorted {
			entries[i] = WinnerEntry{Name: r.Name, Score: r.NsOp}
		}
		winners, runnerUp := FormatWinners(entries)

		if len(winners) > 1 {
			w("\n  winners (tie): %s", strings.Join(winners, ", "))
		} else {
			w("\n  winner: %s", winners[0])
		}
		if runnerUp != nil {
			pct := (runnerUp.Score - entries[0].Score) / entries[0].Score * 100
			w(" (+%.3f%% vs %s)", pct, runnerUp.Name)
		}
		w("\n")
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
	if len(sorted) >= 1 {
		entries := make([]WinnerEntry, len(sorted))
		for i, r := range sorted {
			entries[i] = WinnerEntry{Name: r.Name, Score: avgQPS(r)}
		}
		winners, runnerUp := FormatWinners(entries)

		if len(winners) > 1 {
			w("\n  winners (tie): %s", strings.Join(winners, ", "))
		} else {
			w("\n  winner: %s", winners[0])
		}
		if runnerUp != nil {
			pct := (entries[0].Score - runnerUp.Score) / runnerUp.Score * 100
			w(" (+%.3f%% vs %s)", pct, runnerUp.Name)
		}
		w("\n")
	}
	w("\n")
}

func writeMemoryMarkdown(w func(string, ...any), data []benchmark.MemoryResult) {
	if len(data) == 0 {
		return
	}

	// Show baseline for reference
	if len(data) > 0 && data[0].BaselineBytes > 0 {
		baselineMB := float64(data[0].BaselineBytes) / 1024 / 1024
		w("Baseline (map[string][]byte): %.2f MB\n\n", baselineMB)
	}

	w("| Cache         | Items Stored | Memory (MB) | Overhead vs map (bytes/item) |\n")
	w("|---------------|--------------|-------------|------------------------------|\n")

	for _, r := range data {
		mb := float64(r.Bytes) / 1024 / 1024
		w("| %-13s | %12d | %11.2f | %28d |\n", r.Name, r.Items, mb, r.BytesPerItem)
	}

	// Winner line (lowest memory usage)
	if len(data) >= 1 {
		entries := make([]WinnerEntry, len(data))
		for i, r := range data {
			entries[i] = WinnerEntry{Name: r.Name, Score: float64(r.Bytes)}
		}
		winners, runnerUp := FormatWinners(entries)

		if len(winners) > 1 {
			w("\n  winners (tie): %s", strings.Join(winners, ", "))
		} else {
			w("\n  winner: %s", winners[0])
		}
		if runnerUp != nil {
			pct := (runnerUp.Score - entries[0].Score) / entries[0].Score * 100
			w(" (+%.3f%% vs %s)", pct, runnerUp.Name)
		}
		w("\n")
	}
	w("\n")
}
