// gocachemark is a user-friendly tool for benchmarking Go cache implementations.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tstromberg/gocachemark/internal/benchmark"
	"github.com/tstromberg/gocachemark/internal/cache"
	"github.com/tstromberg/gocachemark/internal/output"
	"github.com/tstromberg/gocachemark/internal/trace"
)

// testFilter holds which hit rate tests to run.
var testFilter map[string]bool

// cacheSizes holds the cache sizes to benchmark.
var cacheSizes []int

// threadCounts holds the thread counts for throughput benchmarks.
var threadCounts []int

func main() {
	hitRate := flag.Bool("hitrate", false, "Run hit rate benchmarks (CDN, Meta, Zipf traces)")
	latency := flag.Bool("latency", false, "Run single-threaded latency benchmarks (ns/op)")
	throughput := flag.Bool("throughput", false, "Run multi-threaded throughput benchmarks (QPS)")
	memory := flag.Bool("memory", false, "Run memory overhead benchmarks (isolated processes)")
	all := flag.Bool("all", false, "Run all benchmarks")
	htmlOut := flag.String("html", "", "Output results to HTML file (e.g., results.html)")
	caches := flag.String("caches", "", "Comma-separated list of caches to benchmark (default: all)")
	tests := flag.String("tests", "", "Comma-separated list of hit rate tests: cdn,meta,zipf (default: all)")
	sizes := flag.String("sizes", "", "Comma-separated cache sizes in K (e.g., 16,32,64,128,256)")
	threads := flag.String("threads", "", "Comma-separated thread counts for throughput (e.g., 8,16)")
	flag.Parse()

	if !*hitRate && !*latency && !*throughput && !*memory && !*all {
		printUsage()
		os.Exit(0)
	}

	// Apply cache filter
	if *caches != "" {
		names := strings.Split(*caches, ",")
		for i, name := range names {
			names[i] = strings.TrimSpace(name)
		}
		cache.SetFilter(names)
	}

	// Apply test filter
	if *tests != "" {
		testFilter = make(map[string]bool)
		for _, t := range strings.Split(*tests, ",") {
			testFilter[strings.TrimSpace(strings.ToLower(t))] = true
		}
	}

	// Apply cache sizes
	cacheSizes = benchmark.DefaultCacheSizes
	if *sizes != "" {
		cacheSizes = nil
		for _, s := range strings.Split(*sizes, ",") {
			s = strings.TrimSpace(s)
			var size int
			if _, err := fmt.Sscanf(s, "%d", &size); err == nil {
				cacheSizes = append(cacheSizes, size*1024)
			}
		}
	}

	// Apply thread counts
	threadCounts = benchmark.DefaultThreadCounts
	if *threads != "" {
		threadCounts = nil
		for _, t := range strings.Split(*threads, ",") {
			t = strings.TrimSpace(t)
			var count int
			if _, err := fmt.Sscanf(t, "%d", &count); err == nil {
				threadCounts = append(threadCounts, count)
			}
		}
	}

	printHeader()

	var results output.Results

	if *hitRate || *all {
		results.HitRate = runHitRateBenchmarks()
	}

	if *latency || *all {
		results.Latency = runLatencyBenchmarks()
	}

	if *throughput || *all {
		results.Throughput = runThroughputBenchmarks()
	}

	if *memory || *all {
		results.Memory = runMemoryBenchmarks()
	}

	results.Rankings = computeOverallRanking(results)
	printOverallRanking(results.Rankings)

	htmlPath := *htmlOut
	if htmlPath == "" {
		htmlPath = filepath.Join(os.TempDir(), "gocachemark-results.html")
	}
	if err := output.WriteHTML(htmlPath, results); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing HTML: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Results: %s\n", htmlPath)
}

func printUsage() {
	fmt.Println("gocachemark - Compare Go cache implementations")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gocachemark -hitrate     Run hit rate benchmarks")
	fmt.Println("  gocachemark -latency     Run single-threaded latency (ns/op)")
	fmt.Println("  gocachemark -throughput  Run multi-threaded throughput (QPS)")
	fmt.Println("  gocachemark -memory      Run memory overhead benchmarks")
	fmt.Println("  gocachemark -all         Run all benchmarks")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -html <file>     Output results to HTML file (default: temp dir)")
	fmt.Println("  -caches <list>   Comma-separated caches to benchmark (default: all)")
	fmt.Println("  -tests <list>    Comma-separated tests to run (default: all)")
	fmt.Println("  -sizes <list>    Comma-separated cache sizes in K (default: 16,32,64,128,256)")
	fmt.Println("  -threads <list>  Comma-separated thread counts for throughput (default: 1,8,16,32)")
	fmt.Println()
	fmt.Println("Available tests:")
	fmt.Println("  Hit rate:    cdn, meta, zipf, twitter, wikipedia")
	fmt.Println("  Latency:     string, int")
	fmt.Println("  Throughput:  string-throughput, int-throughput")
	fmt.Println("  Memory:      memory")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gocachemark -latency -tests int -caches otter,sfcache")
	fmt.Println("  gocachemark -hitrate -tests cdn,zipf")
	fmt.Println("  gocachemark -all -caches otter,theine -html results.html")
	fmt.Println()
	fmt.Println("Available caches:")
	for _, name := range cache.AvailableNames() {
		fmt.Printf("  - %s\n", name)
	}
}

func printHeader() {
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("                       Go Cache Implementation Benchmark")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()
	fmt.Printf("Comparing %d cache implementations.\n", len(cache.AllNames()))
	fmt.Printf("Cache sizes: ")
	for i, size := range cacheSizes {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%dK", size/1024)
	}
	fmt.Println()
	fmt.Println()
}

func shouldRunTest(name string) bool {
	if testFilter == nil {
		return true
	}
	return testFilter[name]
}

func runHitRateBenchmarks() *output.HitRateData {
	sizes := cacheSizes
	data := &output.HitRateData{Sizes: sizes}

	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println("HIT RATE BENCHMARKS")
	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println()

	// CDN Trace
	if shouldRunTest("cdn") {
		fmt.Printf("### [cdn] %s\n\n", trace.CDNInfo())
		cdnResults, err := benchmark.RunCDNHitRate(sizes)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
		} else {
			data.CDN = cdnResults
			printHitRateTable(cdnResults, sizes)
		}
	}

	// Meta Trace
	if shouldRunTest("meta") {
		fmt.Printf("### [meta] %s\n\n", trace.MetaInfo())
		metaResults, err := benchmark.RunMetaHitRate(sizes)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
		} else {
			data.Meta = metaResults
			printHitRateTable(metaResults, sizes)
		}
	}

	// Zipf synthetic
	if shouldRunTest("zipf") {
		fmt.Println("### [zipf] Zipf synthetic trace (alpha=0.8, 2M ops, 100K keyspace)")
		fmt.Println()
		zipfResults := benchmark.RunZipfHitRate(sizes, 100_000, 2_000_000, 0.8)
		data.Zipf = zipfResults
		printHitRateTable(zipfResults, sizes)
	}

	// Twitter Trace
	if shouldRunTest("twitter") {
		fmt.Printf("### [twitter] %s\n\n", trace.TwitterInfo())
		twitterResults, err := benchmark.RunTwitterHitRate(sizes)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
		} else {
			data.Twitter = twitterResults
			printHitRateTable(twitterResults, sizes)
		}
	}

	// Wikipedia Trace
	if shouldRunTest("wikipedia") {
		fmt.Printf("### [wikipedia] %s\n\n", trace.WikipediaInfo())
		wikipediaResults, err := benchmark.RunWikipediaHitRate(sizes)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
		} else {
			data.Wikipedia = wikipediaResults
			printHitRateTable(wikipediaResults, sizes)
		}
	}

	return data
}

func avgHitRate(r benchmark.HitRateResult, sizes []int) float64 {
	var sum float64
	for _, size := range sizes {
		sum += r.Rates[size]
	}
	return sum / float64(len(sizes))
}

func printHitRateTable(results []benchmark.HitRateResult, sizes []int) {
	sorted := make([]benchmark.HitRateResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return avgHitRate(sorted[i], sizes) > avgHitRate(sorted[j], sizes)
	})

	fmt.Print("| Cache         |")
	for _, size := range sizes {
		fmt.Printf(" %5dK |", size/1024)
	}
	fmt.Println("    Avg |")

	fmt.Print("|---------------|")
	for range sizes {
		fmt.Print("--------|")
	}
	fmt.Println("--------|")

	for _, r := range sorted {
		fmt.Printf("| %-13s |", r.Name)
		for _, size := range sizes {
			fmt.Printf(" %5.2f%% |", r.Rates[size])
		}
		fmt.Printf(" %5.2f%% |\n", avgHitRate(r, sizes))
	}
	fmt.Println()

	if len(sorted) >= 2 {
		best := sorted[0]
		second := sorted[1]
		bestAvg := avgHitRate(best, sizes)
		secondAvg := avgHitRate(second, sizes)
		pct := (bestAvg - secondAvg) / secondAvg * 100
		fmt.Printf("Best: %s (%.2f%% avg, %.2f%% better than %s)\n\n", best.Name, bestAvg, pct, second.Name)
	}
}

func runLatencyBenchmarks() *output.LatencyData {
	fmt.Println("-" + strings.Repeat("-", 99))
	fmt.Println("LATENCY BENCHMARKS (Single-Threaded)")
	fmt.Println("-" + strings.Repeat("-", 99))
	fmt.Println()

	data := &output.LatencyData{}

	// String key benchmarks
	if shouldRunTest("string") {
		fmt.Println("### [string] String Keys")
		fmt.Println()
		results := benchmark.RunLatency()
		data.Results = results

		avgLatency := func(r benchmark.LatencyResult) float64 {
			return (r.GetNsOp + r.SetNsOp) / 2
		}

		sorted := make([]benchmark.LatencyResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			return avgLatency(sorted[i]) < avgLatency(sorted[j])
		})

		fmt.Println("| Cache         | Get ns | Get alloc | Set ns | Set alloc | SetEvict ns | SetEvict alloc | Avg ns |")
		fmt.Println("|---------------|--------|-----------|--------|-----------|-------------|----------------|--------|")

		for _, r := range sorted {
			fmt.Printf("| %-13s | %6.0f | %9d | %6.0f | %9d | %11.0f | %14d | %6.0f |\n",
				r.Name, r.GetNsOp, r.GetAllocs, r.SetNsOp, r.SetAllocs, r.SetEvictNsOp, r.SetEvictAllocs, avgLatency(r))
		}
		fmt.Println()

		if len(sorted) >= 2 {
			best := sorted[0]
			second := sorted[1]
			pct := (avgLatency(second) - avgLatency(best)) / avgLatency(best) * 100
			fmt.Printf("Best: %s (%.0f ns avg, %.1f%% faster than %s)\n\n", best.Name, avgLatency(best), pct, second.Name)
		}
	}

	// Int key benchmarks
	if shouldRunTest("int") {
		fmt.Println("### [int] Int Keys")
		fmt.Println()
		intResults := benchmark.RunIntLatency()
		data.IntResults = intResults

		avgIntLatency := func(r benchmark.IntLatencyResult) float64 {
			return (r.GetNsOp + r.SetNsOp) / 2
		}

		intSorted := make([]benchmark.IntLatencyResult, len(intResults))
		copy(intSorted, intResults)
		sort.Slice(intSorted, func(i, j int) bool {
			return avgIntLatency(intSorted[i]) < avgIntLatency(intSorted[j])
		})

		fmt.Println("| Cache         | Get ns | Get alloc | Set ns | Set alloc | Avg ns |")
		fmt.Println("|---------------|--------|-----------|--------|-----------|--------|")

		for _, r := range intSorted {
			fmt.Printf("| %-13s | %6.0f | %9d | %6.0f | %9d | %6.0f |\n",
				r.Name, r.GetNsOp, r.GetAllocs, r.SetNsOp, r.SetAllocs, avgIntLatency(r))
		}
		fmt.Println()

		if len(intSorted) >= 2 {
			best := intSorted[0]
			second := intSorted[1]
			pct := (avgIntLatency(second) - avgIntLatency(best)) / avgIntLatency(best) * 100
			fmt.Printf("Best: %s (%.0f ns avg, %.1f%% faster than %s)\n\n", best.Name, avgIntLatency(best), pct, second.Name)
		}
	}

	return data
}

func runThroughputBenchmarks() *output.ThroughputData {
	threads := threadCounts

	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println("THROUGHPUT BENCHMARKS (Multi-Threaded)")
	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println()

	data := &output.ThroughputData{Threads: threads}

	avgQPS := func(r benchmark.ThroughputResult) float64 {
		var sum float64
		for _, t := range threads {
			sum += r.QPS[t]
		}
		return sum / float64(len(threads))
	}

	printThroughputTable := func(results []benchmark.ThroughputResult) {
		sorted := make([]benchmark.ThroughputResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			return avgQPS(sorted[i]) > avgQPS(sorted[j])
		})

		fmt.Print("| Cache         |")
		for _, t := range threads {
			fmt.Printf(" %2dT       |", t)
		}
		fmt.Println("       Avg |")

		fmt.Print("|---------------|")
		for range threads {
			fmt.Print("-----------|")
		}
		fmt.Println("-----------|")

		for _, r := range sorted {
			fmt.Printf("| %-13s |", r.Name)
			for _, t := range threads {
				qps := r.QPS[t]
				if qps >= 1_000_000 {
					fmt.Printf(" %6.2fM   |", qps/1_000_000)
				} else {
					fmt.Printf(" %6.0fK   |", qps/1_000)
				}
			}
			avg := avgQPS(r)
			if avg >= 1_000_000 {
				fmt.Printf(" %6.2fM   |\n", avg/1_000_000)
			} else {
				fmt.Printf(" %6.0fK   |\n", avg/1_000)
			}
		}
		fmt.Println()

		if len(sorted) >= 2 {
			best := sorted[0]
			second := sorted[1]
			bestAvg := avgQPS(best)
			secondAvg := avgQPS(second)
			pct := (bestAvg - secondAvg) / secondAvg * 100
			fmt.Printf("Best: %s (%.1f%% faster than %s on average)\n\n", best.Name, pct, second.Name)
		}
	}

	// String key throughput
	if shouldRunTest("string-throughput") {
		fmt.Println("### [string-throughput] String keys, Zipf workload, 75% reads / 25% writes")
		fmt.Println()
		data.Results = benchmark.RunThroughput(threads)
		printThroughputTable(data.Results)
	}

	// Int key throughput
	if shouldRunTest("int-throughput") {
		fmt.Println("### [int-throughput] Int keys, Zipf workload, 75% reads / 25% writes")
		fmt.Println()
		data.IntResults = benchmark.RunIntThroughput(threads)
		printThroughputTable(data.IntResults)
	}

	return data
}

func runMemoryBenchmarks() *output.MemoryData {
	capacity := benchmark.DefaultMemoryCapacity
	valSize := benchmark.DefaultValueSize

	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println("MEMORY BENCHMARKS (Isolated Processes)")
	fmt.Println("-" + strings.Repeat("-", 79))
	fmt.Println()

	if !shouldRunTest("memory") {
		return nil
	}

	fmt.Printf("### [memory] %d items, %d byte values, 3 passes for admission\n\n", capacity, valSize)

	results, err := benchmark.RunMemory(capacity, valSize)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		return nil
	}

	fmt.Println("| Cache         | Items Stored | Memory (MB) | Overhead vs map (bytes/item) |")
	fmt.Println("|---------------|--------------|-------------|------------------------------|")

	for _, r := range results {
		mb := float64(r.Bytes) / 1024 / 1024
		fmt.Printf("| %-13s | %12d | %9.2f MB | %28d |\n",
			r.Name, r.Items, mb, r.BytesPerItem)
	}
	fmt.Println()

	if len(results) >= 2 {
		best := results[0]
		second := results[1]
		savings := float64(second.Bytes-best.Bytes) / float64(second.Bytes) * 100
		fmt.Printf("Most efficient: %s (%.1f%% less memory than %s)\n\n", best.Name, savings, second.Name)
	}

	return &output.MemoryData{Results: results, Capacity: capacity, ValSize: valSize}
}

func computeOverallRanking(results output.Results) []output.Ranking {
	scores := make(map[string]float64)

	// Assign points based on ranking position in each test
	// Points: 1st=10, 2nd=7, 3rd=5, 4th=4, 5th=3, 6th=2, 7th=1, rest=0
	assignPoints := func(names []string) {
		points := []float64{10, 7, 5, 4, 3, 2, 1}
		for i, name := range names {
			if i < len(points) {
				scores[name] += points[i]
			}
		}
	}

	// Hit rate benchmarks - rank by average hit rate (higher is better)
	if results.HitRate != nil {
		if results.HitRate.CDN != nil && len(results.HitRate.CDN) > 0 {
			sorted := make([]benchmark.HitRateResult, len(results.HitRate.CDN))
			copy(sorted, results.HitRate.CDN)
			sort.Slice(sorted, func(i, j int) bool {
				return avgHitRate(sorted[i], results.HitRate.Sizes) > avgHitRate(sorted[j], results.HitRate.Sizes)
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
		if results.HitRate.Meta != nil && len(results.HitRate.Meta) > 0 {
			sorted := make([]benchmark.HitRateResult, len(results.HitRate.Meta))
			copy(sorted, results.HitRate.Meta)
			sort.Slice(sorted, func(i, j int) bool {
				return avgHitRate(sorted[i], results.HitRate.Sizes) > avgHitRate(sorted[j], results.HitRate.Sizes)
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
		if results.HitRate.Zipf != nil && len(results.HitRate.Zipf) > 0 {
			sorted := make([]benchmark.HitRateResult, len(results.HitRate.Zipf))
			copy(sorted, results.HitRate.Zipf)
			sort.Slice(sorted, func(i, j int) bool {
				return avgHitRate(sorted[i], results.HitRate.Sizes) > avgHitRate(sorted[j], results.HitRate.Sizes)
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
		if results.HitRate.Twitter != nil && len(results.HitRate.Twitter) > 0 {
			sorted := make([]benchmark.HitRateResult, len(results.HitRate.Twitter))
			copy(sorted, results.HitRate.Twitter)
			sort.Slice(sorted, func(i, j int) bool {
				return avgHitRate(sorted[i], results.HitRate.Sizes) > avgHitRate(sorted[j], results.HitRate.Sizes)
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
		if results.HitRate.Wikipedia != nil && len(results.HitRate.Wikipedia) > 0 {
			sorted := make([]benchmark.HitRateResult, len(results.HitRate.Wikipedia))
			copy(sorted, results.HitRate.Wikipedia)
			sort.Slice(sorted, func(i, j int) bool {
				return avgHitRate(sorted[i], results.HitRate.Sizes) > avgHitRate(sorted[j], results.HitRate.Sizes)
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
	}

	// Latency benchmarks - rank by average latency (lower is better)
	if results.Latency != nil {
		if results.Latency.Results != nil && len(results.Latency.Results) > 0 {
			avgLatency := func(r benchmark.LatencyResult) float64 {
				return (r.GetNsOp + r.SetNsOp) / 2
			}
			sorted := make([]benchmark.LatencyResult, len(results.Latency.Results))
			copy(sorted, results.Latency.Results)
			sort.Slice(sorted, func(i, j int) bool {
				return avgLatency(sorted[i]) < avgLatency(sorted[j])
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
		if results.Latency.IntResults != nil && len(results.Latency.IntResults) > 0 {
			avgIntLatency := func(r benchmark.IntLatencyResult) float64 {
				return (r.GetNsOp + r.SetNsOp) / 2
			}
			sorted := make([]benchmark.IntLatencyResult, len(results.Latency.IntResults))
			copy(sorted, results.Latency.IntResults)
			sort.Slice(sorted, func(i, j int) bool {
				return avgIntLatency(sorted[i]) < avgIntLatency(sorted[j])
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
	}

	// Throughput benchmarks - rank by average QPS (higher is better)
	if results.Throughput != nil {
		avgQPS := func(r benchmark.ThroughputResult) float64 {
			var sum float64
			for _, qps := range r.QPS {
				sum += qps
			}
			return sum / float64(len(r.QPS))
		}

		if results.Throughput.Results != nil && len(results.Throughput.Results) > 0 {
			sorted := make([]benchmark.ThroughputResult, len(results.Throughput.Results))
			copy(sorted, results.Throughput.Results)
			sort.Slice(sorted, func(i, j int) bool {
				return avgQPS(sorted[i]) > avgQPS(sorted[j])
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
		if results.Throughput.IntResults != nil && len(results.Throughput.IntResults) > 0 {
			sorted := make([]benchmark.ThroughputResult, len(results.Throughput.IntResults))
			copy(sorted, results.Throughput.IntResults)
			sort.Slice(sorted, func(i, j int) bool {
				return avgQPS(sorted[i]) > avgQPS(sorted[j])
			})
			var names []string
			for _, r := range sorted {
				names = append(names, r.Name)
			}
			assignPoints(names)
		}
	}

	// Memory benchmark - rank by memory usage (lower is better)
	if results.Memory != nil && results.Memory.Results != nil && len(results.Memory.Results) > 0 {
		var names []string
		for _, r := range results.Memory.Results {
			names = append(names, r.Name)
		}
		assignPoints(names)
	}

	// No tests were run
	if len(scores) == 0 {
		return nil
	}

	// Sort caches by score
	type ranking struct {
		name  string
		score float64
	}
	var rankings []ranking
	for name, score := range scores {
		rankings = append(rankings, ranking{name, score})
	}
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].score > rankings[j].score
	})

	// Convert to output.Ranking slice
	var result []output.Ranking
	for i, r := range rankings {
		result = append(result, output.Ranking{
			Rank:  i + 1,
			Name:  r.name,
			Score: r.score,
		})
	}
	return result
}

func printOverallRanking(rankings []output.Ranking) {
	if len(rankings) == 0 {
		return
	}

	// Print top 3
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("OVERALL RANKING (ranked voting across all tests)")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()

	medals := []string{"🥇", "🥈", "🥉"}
	for i := 0; i < len(rankings) && i < 3; i++ {
		r := rankings[i]
		medal := medals[i]
		fmt.Printf("%s #%d: %s (%.0f points)\n", medal, r.Rank, r.Name, r.Score)
	}
	fmt.Println()
}
