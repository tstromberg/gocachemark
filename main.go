// gocachemark is a user-friendly tool for benchmarking Go cache implementations.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// validSuites lists all available benchmark suites.
var validSuites = []string{"hitrate", "latency", "throughput", "memory"}

// validTests lists all available test names.
var validTests = []string{
	// hitrate
	"cdn", "meta", "zipf", "twitter", "wikipedia", "thesios-block", "thesios-file", "ibm-docker", "tencent-photo",
	// latency
	"string", "int", "getorset",
	// throughput
	"string-throughput", "int-throughput", "getorset-throughput",
	// memory
	"memory",
}

// suiteFilter holds which suites to run.
var suiteFilter map[string]bool

func main() {
	showHelp := flag.Bool("help", false, "Show help message")
	suites := flag.String("suites", "all", "Comma-separated list of benchmark suites: hitrate,latency,throughput,memory (default: all)")
	htmlOut := flag.String("html", "", "Output results to HTML file (e.g., results.html)")
	openHTML := flag.Bool("open", false, "Open HTML report in web browser after generation")
	caches := flag.String("caches", "", "Comma-separated list of caches to benchmark (default: all)")
	tests := flag.String("tests", "", "Comma-separated list of tests to run across suites (default: all)")
	sizes := flag.String("sizes", "", "Comma-separated cache sizes in K (e.g., 16,32,64,128,256)")
	threads := flag.String("threads", "", "Comma-separated thread counts for throughput (e.g., 8,16)")
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	// Parse suites
	suiteFilter = make(map[string]bool)
	if *suites == "all" || *suites == "" {
		for _, s := range validSuites {
			suiteFilter[s] = true
		}
	} else {
		for _, s := range strings.Split(*suites, ",") {
			s = strings.TrimSpace(strings.ToLower(s))
			if s != "" {
				suiteFilter[s] = true
			}
		}
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
		validTestSet := make(map[string]bool)
		for _, t := range validTests {
			validTestSet[t] = true
		}
		for _, t := range strings.Split(*tests, ",") {
			t = strings.TrimSpace(strings.ToLower(t))
			if t == "" {
				continue
			}
			if !validTestSet[t] {
				fmt.Fprintf(os.Stderr, "error: unknown test %q\n\nAvailable tests:\n", t)
				for _, vt := range validTests {
					fmt.Fprintf(os.Stderr, "  %s\n", vt)
				}
				os.Exit(1)
			}
			testFilter[t] = true
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

	if suiteFilter["hitrate"] {
		results.HitRate = runHitRateBenchmarks()
	}

	if suiteFilter["latency"] {
		results.Latency = runLatencyBenchmarks()
	}

	if suiteFilter["throughput"] {
		results.Throughput = runThroughputBenchmarks()
	}

	if suiteFilter["memory"] {
		results.Memory = runMemoryBenchmarks()
	}

	results.Rankings = computeOverallRanking(results)
	printOverallRanking(results.Rankings)

	htmlPath := *htmlOut
	if htmlPath == "" {
		htmlPath = filepath.Join(os.TempDir(), "gocachemark-results.html")
	}

	// Build command line string
	commandLine := "gocachemark " + strings.Join(os.Args[1:], " ")

	if err := output.WriteHTML(htmlPath, results, commandLine); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing HTML: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Results: %s\n", htmlPath)

	if *openHTML {
		if err := openBrowser(htmlPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		}
	}
}

func printUsage() {
	fmt.Println("gocachemark - Compare Go cache implementations")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gocachemark                      Run all benchmarks (default)")
	fmt.Println("  gocachemark -suites hitrate      Run only hit rate benchmarks")
	fmt.Println("  gocachemark -suites latency,memory  Run latency and memory benchmarks")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -suites <list>   Comma-separated suites: hitrate,latency,throughput,memory (default: all)")
	fmt.Println("  -tests <list>    Comma-separated tests to run across suites (default: all)")
	fmt.Println("  -caches <list>   Comma-separated caches to benchmark (default: all)")
	fmt.Println("  -sizes <list>    Comma-separated cache sizes in K (default: 16,32,64,128,256)")
	fmt.Println("  -threads <list>  Comma-separated thread counts for throughput (default: 1,8,16,32)")
	fmt.Println("  -html <file>     Output results to HTML file (default: temp dir)")
	fmt.Println("  -open            Open HTML report in web browser after generation")
	fmt.Println()
	fmt.Println("Available suites and tests:")
	fmt.Println()
	fmt.Println("  hitrate - Hit rate benchmarks (cache efficiency)")
	fmt.Println("    cdn                     CDN access trace")
	fmt.Println("    meta                    Meta/Facebook KV trace")
	fmt.Println("    zipf                    Synthetic Zipf distribution")
	fmt.Println("    twitter                 Twitter cache trace")
	fmt.Println("    wikipedia               Wikipedia access trace")
	fmt.Println("    thesios-block           Google Thesios I/O block trace")
	fmt.Println("    thesios-file            Google Thesios I/O file trace")
	fmt.Println("    ibm-docker              IBM Docker Registry trace")
	fmt.Println("    tencent-photo           Tencent Photo trace")
	fmt.Println()
	fmt.Println("  latency - Single-threaded latency benchmarks (ns/op)")
	fmt.Println("    string                  String key Get/Set operations")
	fmt.Println("    int                     Int key Get/Set operations")
	fmt.Println("    getorset                GetOrSet operations (URL keys)")
	fmt.Println()
	fmt.Println("  throughput - Multi-threaded throughput benchmarks (QPS)")
	fmt.Println("    string-throughput       String keys, Zipf workload")
	fmt.Println("    int-throughput          Int keys, Zipf workload")
	fmt.Println("    getorset-throughput     GetOrSet operations (URL keys)")
	fmt.Println()
	fmt.Println("  memory - Memory overhead benchmarks (isolated processes)")
	fmt.Println("    memory                  Per-item memory overhead")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gocachemark -suites latency -tests int -caches otter,multicache")
	fmt.Println("  gocachemark -suites hitrate -tests cdn,zipf")
	fmt.Println("  gocachemark -suites throughput,memory -tests string-throughput,memory")
	fmt.Println("  gocachemark -caches otter,theine -html results.html")
	fmt.Println()
	fmt.Println("Available caches:")
	for _, name := range cache.AvailableNames() {
		fmt.Printf("  - %s\n", name)
	}
}

const lineWidth = 80

func printHeader() {
	fmt.Println("gocachemark")
	fmt.Println()

	// Build config summary
	var suitesRun []string
	for _, s := range validSuites {
		if suiteFilter[s] {
			suitesRun = append(suitesRun, s)
		}
	}

	fmt.Printf("  caches: %d\n", len(cache.AllNames()))
	fmt.Printf("  suites: %s\n", strings.Join(suitesRun, ", "))

	var sizeStrs []string
	for _, size := range cacheSizes {
		sizeStrs = append(sizeStrs, fmt.Sprintf("%dK", size/1024))
	}
	fmt.Printf("  sizes:  %s\n", strings.Join(sizeStrs, ", "))
	fmt.Println()
}

func printSuite(name, description string) {
	header := fmt.Sprintf("%s: %s ", name, description)
	padding := lineWidth - len(header)
	if padding < 4 {
		padding = 4
	}
	fmt.Printf("%s%s\n\n", header, strings.Repeat("─", padding))
}

func printTest(name, description string) {
	fmt.Printf("  [%s] %s\n\n", name, description)
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

	printSuite("hitrate", "cache efficiency")

	if shouldRunTest("cdn") {
		printTest("cdn", trace.CDNInfo())
		cdnResults, err := benchmark.RunCDNHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.CDN = cdnResults
			printHitRateTable(cdnResults, sizes)
		}
	}

	if shouldRunTest("meta") {
		printTest("meta", trace.MetaInfo())
		metaResults, err := benchmark.RunMetaHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.Meta = metaResults
			printHitRateTable(metaResults, sizes)
		}
	}

	if shouldRunTest("zipf") {
		printTest("zipf", "Zipf synthetic (alpha=0.8, 2M ops, 100K keyspace)")
		zipfResults := benchmark.RunZipfHitRate(sizes, 100_000, 2_000_000, 0.8)
		data.Zipf = zipfResults
		printHitRateTable(zipfResults, sizes)
	}

	if shouldRunTest("twitter") {
		printTest("twitter", trace.TwitterInfo())
		twitterResults, err := benchmark.RunTwitterHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.Twitter = twitterResults
			printHitRateTable(twitterResults, sizes)
		}
	}

	if shouldRunTest("wikipedia") {
		printTest("wikipedia", trace.WikipediaInfo())
		wikipediaResults, err := benchmark.RunWikipediaHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.Wikipedia = wikipediaResults
			printHitRateTable(wikipediaResults, sizes)
		}
	}

	if shouldRunTest("thesios-block") {
		printTest("thesios-block", trace.ThesiosBlockInfo())
		thesiosBlockResults, err := benchmark.RunThesiosBlockHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.ThesiosBlock = thesiosBlockResults
			printHitRateTable(thesiosBlockResults, sizes)
		}
	}

	if shouldRunTest("thesios-file") {
		printTest("thesios-file", trace.ThesiosFileInfo())
		thesiosFileResults, err := benchmark.RunThesiosFileHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.ThesiosFile = thesiosFileResults
			printHitRateTable(thesiosFileResults, sizes)
		}
	}

	if shouldRunTest("ibm-docker") {
		printTest("ibm-docker", trace.IBMDockerInfo())
		ibmDockerResults, err := benchmark.RunIBMDockerHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.IBMDocker = ibmDockerResults
			printHitRateTable(ibmDockerResults, sizes)
		}
	}

	if shouldRunTest("tencent-photo") {
		printTest("tencent-photo", trace.TencentPhotoInfo())
		tencentPhotoResults, err := benchmark.RunTencentPhotoHitRate(sizes)
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
		} else {
			data.TencentPhoto = tencentPhotoResults
			printHitRateTable(tencentPhotoResults, sizes)
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

	fmt.Print("  | Cache         |")
	for _, size := range sizes {
		fmt.Printf(" %5dK |", size/1024)
	}
	fmt.Println("    Avg |")

	fmt.Print("  |---------------|")
	for range sizes {
		fmt.Print("--------|")
	}
	fmt.Println("--------|")

	for _, r := range sorted {
		fmt.Printf("  | %-13s |", r.Name)
		for _, size := range sizes {
			fmt.Printf(" %5.2f%% |", r.Rates[size])
		}
		fmt.Printf(" %5.2f%% |\n", avgHitRate(r, sizes))
	}

	if len(sorted) >= 2 {
		best := sorted[0]
		second := sorted[1]
		bestAvg := avgHitRate(best, sizes)
		secondAvg := avgHitRate(second, sizes)
		pct := (bestAvg - secondAvg) / secondAvg * 100
		fmt.Printf("\n  winner: %s (%.2f%% avg, +%.2f%% vs %s)\n", best.Name, bestAvg, pct, second.Name)
	}
	fmt.Println()
}

func runLatencyBenchmarks() *output.LatencyData {
	printSuite("latency", "single-threaded (ns/op)")

	data := &output.LatencyData{}

	if shouldRunTest("string") {
		printTest("string", "string key Get/Set operations")
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

		fmt.Println("  | Cache         | Get ns | Get alloc | Set ns | Set alloc | SetEvict ns | SetEvict alloc | Avg ns |")
		fmt.Println("  |---------------|--------|-----------|--------|-----------|-------------|----------------|--------|")

		for _, r := range sorted {
			fmt.Printf("  | %-13s | %6.0f | %9d | %6.0f | %9d | %11.0f | %14d | %6.0f |\n",
				r.Name, r.GetNsOp, r.GetAllocs, r.SetNsOp, r.SetAllocs, r.SetEvictNsOp, r.SetEvictAllocs, avgLatency(r))
		}

		if len(sorted) >= 2 {
			best := sorted[0]
			second := sorted[1]
			pct := (avgLatency(second) - avgLatency(best)) / avgLatency(best) * 100
			fmt.Printf("\n  winner: %s (%.0f ns avg, %s is %.1f%% slower)\n", best.Name, avgLatency(best), second.Name, pct)
		}
		fmt.Println()
	}

	if shouldRunTest("getorset") {
		printTest("getorset", "GetOrSet operations (URL keys)")
		results := benchmark.RunGetOrSetLatency()
		data.GetOrSetResults = results

		if len(results) == 0 {
			fmt.Println("  (no caches with GetOrSet support)")
			fmt.Println()
		} else {
			sorted := make([]benchmark.GetOrSetLatencyResult, len(results))
			copy(sorted, results)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].NsOp < sorted[j].NsOp
			})

			fmt.Println("  | Cache         | GetOrSet ns | GetOrSet alloc |")
			fmt.Println("  |---------------|-------------|----------------|")

			for _, r := range sorted {
				fmt.Printf("  | %-13s | %11.0f | %14d |\n", r.Name, r.NsOp, r.Allocs)
			}

			if len(sorted) >= 2 {
				best := sorted[0]
				second := sorted[1]
				pct := (second.NsOp - best.NsOp) / best.NsOp * 100
				fmt.Printf("\n  winner: %s (%.0f ns, %s is %.1f%% slower)\n", best.Name, best.NsOp, second.Name, pct)
			}
			fmt.Println()
		}
	}

	if shouldRunTest("int") {
		printTest("int", "int key Get/Set operations")
		results := benchmark.RunIntLatency()
		data.IntResults = results

		avgLatency := func(r benchmark.IntLatencyResult) float64 {
			return (r.GetNsOp + r.SetNsOp) / 2
		}

		sorted := make([]benchmark.IntLatencyResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			return avgLatency(sorted[i]) < avgLatency(sorted[j])
		})

		fmt.Println("  | Cache         | Get ns | Get alloc | Set ns | Set alloc | Avg ns |")
		fmt.Println("  |---------------|--------|-----------|--------|-----------|--------|")

		for _, r := range sorted {
			fmt.Printf("  | %-13s | %6.0f | %9d | %6.0f | %9d | %6.0f |\n",
				r.Name, r.GetNsOp, r.GetAllocs, r.SetNsOp, r.SetAllocs, avgLatency(r))
		}

		if len(sorted) >= 2 {
			best := sorted[0]
			second := sorted[1]
			pct := (avgLatency(second) - avgLatency(best)) / avgLatency(best) * 100
			fmt.Printf("\n  winner: %s (%.0f ns avg, %s is %.1f%% slower)\n", best.Name, avgLatency(best), second.Name, pct)
		}
		fmt.Println()
	}

	return data
}

func runThroughputBenchmarks() *output.ThroughputData {
	threads := threadCounts

	printSuite("throughput", "multi-threaded (QPS)")

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

		fmt.Print("  | Cache         |")
		for _, t := range threads {
			fmt.Printf(" %2dT       |", t)
		}
		fmt.Println("       Avg |")

		fmt.Print("  |---------------|")
		for range threads {
			fmt.Print("-----------|")
		}
		fmt.Println("-----------|")

		for _, r := range sorted {
			fmt.Printf("  | %-13s |", r.Name)
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

		if len(sorted) >= 2 {
			best := sorted[0]
			second := sorted[1]
			bestAvg := avgQPS(best)
			secondAvg := avgQPS(second)
			pct := (bestAvg - secondAvg) / secondAvg * 100
			fmt.Printf("\n  winner: %s (+%.1f%% vs %s)\n", best.Name, pct, second.Name)
		}
		fmt.Println()
	}

	if shouldRunTest("string-throughput") {
		printTest("string-throughput", "string keys, Zipf, 75% read / 25% write")
		data.Results = benchmark.RunThroughput(threads)
		printThroughputTable(data.Results)
	}

	if shouldRunTest("int-throughput") {
		printTest("int-throughput", "int keys, Zipf, 75% read / 25% write")
		data.IntResults = benchmark.RunIntThroughput(threads)
		printThroughputTable(data.IntResults)
	}

	if shouldRunTest("getorset-throughput") {
		printTest("getorset-throughput", "GetOrSet operations (URL keys)")
		data.GetOrSetResults = benchmark.RunGetOrSetThroughput(threads)
		if len(data.GetOrSetResults) > 0 {
			printThroughputTable(data.GetOrSetResults)
		} else {
			fmt.Println("  (no caches with GetOrSet support)")
			fmt.Println()
		}
	}

	return data
}

func runMemoryBenchmarks() *output.MemoryData {
	capacity := benchmark.DefaultMemoryCapacity
	valSize := benchmark.DefaultValueSize

	printSuite("memory", "overhead per item (isolated processes)")

	if !shouldRunTest("memory") {
		return nil
	}

	printTest("memory", fmt.Sprintf("%d items, %d byte values, 3 passes", capacity, valSize))

	results, err := benchmark.RunMemory(capacity, valSize)
	if err != nil {
		fmt.Printf("  error: %v\n\n", err)
		return nil
	}

	fmt.Println("  | Cache         | Items Stored | Memory (MB) | Overhead (bytes/item) |")
	fmt.Println("  |---------------|--------------|-------------|-----------------------|")

	for _, r := range results {
		mb := float64(r.Bytes) / 1024 / 1024
		fmt.Printf("  | %-13s | %12d | %11.2f | %21d |\n",
			r.Name, r.Items, mb, r.BytesPerItem)
	}

	if len(results) >= 2 {
		best := results[0]
		second := results[1]
		savings := float64(second.Bytes-best.Bytes) / float64(second.Bytes) * 100
		fmt.Printf("\n  winner: %s (%.1f%% less memory vs %s)\n", best.Name, savings, second.Name)
	}
	fmt.Println()

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
		hitRateBenchmarks := [][]benchmark.HitRateResult{
			results.HitRate.CDN,
			results.HitRate.Meta,
			results.HitRate.Zipf,
			results.HitRate.Twitter,
			results.HitRate.Wikipedia,
			results.HitRate.ThesiosBlock,
			results.HitRate.ThesiosFile,
			results.HitRate.IBMDocker,
			results.HitRate.TencentPhoto,
		}
		for _, data := range hitRateBenchmarks {
			if len(data) == 0 {
				continue
			}
			sorted := make([]benchmark.HitRateResult, len(data))
			copy(sorted, data)
			sort.Slice(sorted, func(i, j int) bool {
				return avgHitRate(sorted[i], results.HitRate.Sizes) > avgHitRate(sorted[j], results.HitRate.Sizes)
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
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
		// GetOrSet latency rankings
		if len(results.Latency.GetOrSetResults) > 0 {
			sorted := make([]benchmark.GetOrSetLatencyResult, len(results.Latency.GetOrSetResults))
			copy(sorted, results.Latency.GetOrSetResults)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].NsOp < sorted[j].NsOp
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
		throughputBenchmarks := [][]benchmark.ThroughputResult{
			results.Throughput.Results,
			results.Throughput.IntResults,
			results.Throughput.GetOrSetResults,
		}
		for _, data := range throughputBenchmarks {
			if len(data) == 0 {
				continue
			}
			sorted := make([]benchmark.ThroughputResult, len(data))
			copy(sorted, data)
			sort.Slice(sorted, func(i, j int) bool {
				return avgQPS(sorted[i]) > avgQPS(sorted[j])
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
			}
			assignPoints(names)
		}
	}

	// Memory benchmark - rank by memory usage (lower is better)
	if results.Memory != nil && len(results.Memory.Results) > 0 {
		names := make([]string, len(results.Memory.Results))
		for i, r := range results.Memory.Results {
			names[i] = r.Name
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

	printSuite("summary", "ranked voting across all tests")

	for i := 0; i < len(rankings) && i < 3; i++ {
		r := rankings[i]
		fmt.Printf("  #%d  %s (%.0f points)\n", r.Rank, r.Name, r.Score)
	}
	fmt.Println()
}

// openBrowser opens the specified path in the default web browser.
func openBrowser(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
