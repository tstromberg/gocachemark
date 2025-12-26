// Package output provides result formatting and export.
package output

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

//go:embed template.html
var templateFS embed.FS

// Results holds all benchmark results for HTML output.
type Results struct {
	Timestamp   string
	HitRate     *HitRateData
	Latency     *LatencyData
	Throughput  *ThroughputData
	Memory      *MemoryData
	Rankings    []Ranking
	MedalTable  *MedalTable
	MachineInfo MachineInfo
}

// MachineInfo holds information about the benchmark environment.
type MachineInfo struct {
	OS          string
	Arch        string
	NumCPU      int
	GoVersion   string
	CommandLine string
}

// Ranking represents an overall ranking entry.
type Ranking struct {
	Rank   int
	Name   string
	Score  float64
	Gold   int
	Silver int
	Bronze int
}

// BenchmarkMedal represents a single benchmark's top 3 placements.
type BenchmarkMedal struct {
	Name   string
	Gold   string
	Silver string
	Bronze string
}

// CategoryMedals holds medals for a benchmark category with its winner.
type CategoryMedals struct {
	Name       string
	Benchmarks []BenchmarkMedal
	Rankings   []Ranking
}

// MedalTable holds all benchmark medals organized by category.
type MedalTable struct {
	Categories []CategoryMedals
}

// MemoryData holds memory benchmark data.
type MemoryData struct {
	Results  []benchmark.MemoryResult
	Capacity int
	ValSize  int
}

// HitRateData holds hit rate benchmark data.
type HitRateData struct {
	CDN          []benchmark.HitRateResult
	Meta         []benchmark.HitRateResult
	Zipf         []benchmark.HitRateResult
	Twitter      []benchmark.HitRateResult
	Wikipedia    []benchmark.HitRateResult
	ThesiosBlock []benchmark.HitRateResult
	ThesiosFile  []benchmark.HitRateResult
	IBMDocker    []benchmark.HitRateResult
	TencentPhoto []benchmark.HitRateResult
	Sizes        []int
}

// LatencyData holds latency benchmark data.
type LatencyData struct {
	Results         []benchmark.LatencyResult
	IntResults      []benchmark.IntLatencyResult
	GetOrSetResults []benchmark.GetOrSetLatencyResult
}

// ThroughputData holds throughput benchmark data.
type ThroughputData struct {
	StringGetResults []benchmark.ThroughputResult
	StringSetResults []benchmark.ThroughputResult
	IntGetResults    []benchmark.ThroughputResult
	IntSetResults    []benchmark.ThroughputResult
	GetOrSetResults  []benchmark.ThroughputResult
	Threads          []int
}

// WriteHTML writes benchmark results to an HTML file.
func WriteHTML(filename string, results Results, commandLine string) error {
	results.Timestamp = time.Now().Format("2006-01-02 15:04:05 MST")
	results.MachineInfo.CommandLine = commandLine

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return htmlTemplate.Execute(f, results)
}

var htmlTemplate = template.Must(template.New("template.html").Funcs(templateFuncs).ParseFS(templateFS, "template.html"))

var templateFuncs = template.FuncMap{
	"add":  func(a, b int) int { return a + b },
	"addf": func(a, b float64) float64 { return a + b },
	"sub":  func(a, b float64) float64 { return a - b },
	"mul":  func(a, b float64) float64 { return a * b },
	"div": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"avgHitRate": func(r benchmark.HitRateResult, sizes []int) float64 {
		var sum float64
		for _, size := range sizes {
			sum += r.Rates[size]
		}
		return sum / float64(len(sizes))
	},
	"avgLatency": func(r benchmark.LatencyResult) float64 {
		return (r.GetNsOp + r.SetNsOp) / 2
	},
	"avgIntLatency": func(r benchmark.IntLatencyResult) float64 {
		return (r.GetNsOp + r.SetNsOp) / 2
	},
	"avgQPS": func(r benchmark.ThroughputResult) float64 {
		var sum float64
		for _, qps := range r.QPS {
			sum += qps
		}
		return sum / float64(len(r.QPS))
	},
	"sortByHitRate": func(results []benchmark.HitRateResult, sizes []int) []benchmark.HitRateResult {
		sorted := make([]benchmark.HitRateResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			var sumI, sumJ float64
			for _, size := range sizes {
				sumI += sorted[i].Rates[size]
				sumJ += sorted[j].Rates[size]
			}
			return sumI > sumJ
		})
		return sorted
	},
	"sortByGetLatency": func(results []benchmark.LatencyResult) []benchmark.LatencyResult {
		sorted := make([]benchmark.LatencyResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].GetNsOp < sorted[j].GetNsOp
		})
		return sorted
	},
	"sortByIntGetLatency": func(results []benchmark.IntLatencyResult) []benchmark.IntLatencyResult {
		sorted := make([]benchmark.IntLatencyResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].GetNsOp < sorted[j].GetNsOp
		})
		return sorted
	},
	"sortByThroughput": func(results []benchmark.ThroughputResult, threads []int) []benchmark.ThroughputResult {
		sorted := make([]benchmark.ThroughputResult, len(results))
		copy(sorted, results)
		maxThreads := threads[len(threads)-1]
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].QPS[maxThreads] > sorted[j].QPS[maxThreads]
		})
		return sorted
	},
	"divK": func(n int) int { return n / 1024 },
	"sizeLabels": func(sizes []int) template.JS {
		labels := make([]string, len(sizes))
		for i, s := range sizes {
			labels[i] = fmt.Sprintf("\"%dK\"", s/1024)
		}
		return template.JS("[" + strings.Join(labels, ",") + "]")
	},
	"hitRateDatasets": func(results []benchmark.HitRateResult, sizes []int) template.JS {
		fallback := []string{"#388E3C", "#1E88E5", "#E53935", "#8E24AA", "#FB8C00"}
		var datasets []string
		for i, r := range results {
			color, ok := cacheColors[r.Name]
			if !ok {
				color = fallback[i%len(fallback)]
			}
			var data []string
			for _, size := range sizes {
				data = append(data, fmt.Sprintf("%.2f", r.Rates[size]))
			}
			datasets = append(datasets, fmt.Sprintf(
				`{label:"%s",data:[%s],borderColor:"%s",backgroundColor:"%s",tension:0.1,fill:false,borderWidth:1.5,pointRadius:2,pointHoverRadius:4}`,
				r.Name, strings.Join(data, ","), color, color,
			))
		}
		return template.JS("[" + strings.Join(datasets, ",") + "]")
	},
	"threadLabels": func(threads []int) template.JS {
		labels := make([]string, len(threads))
		for i, t := range threads {
			labels[i] = fmt.Sprintf("\"%dT\"", t)
		}
		return template.JS("[" + strings.Join(labels, ",") + "]")
	},
	"throughputDatasets": func(results []benchmark.ThroughputResult, threads []int) template.JS {
		fallback := []string{"#388E3C", "#1E88E5", "#E53935", "#8E24AA", "#FB8C00"}
		var datasets []string
		for i, r := range results {
			color, ok := cacheColors[r.Name]
			if !ok {
				color = fallback[i%len(fallback)]
			}
			var data []string
			for _, t := range threads {
				data = append(data, fmt.Sprintf("%.0f", r.QPS[t]))
			}
			datasets = append(datasets, fmt.Sprintf(
				`{label:"%s",data:[%s],borderColor:"%s",backgroundColor:"%s",tension:0.1,fill:false,borderWidth:1.5,pointRadius:2,pointHoverRadius:4}`,
				r.Name, strings.Join(data, ","), color, color,
			))
		}
		return template.JS("[" + strings.Join(datasets, ",") + "]")
	},
	"allocColor": func(n int64) template.CSS {
		switch {
		case n == 0:
			return "background:#fff;color:#333"
		case n == 1:
			return "background:#fff3cd;color:#333"
		case n == 2:
			return "background:#ffcc80;color:#333"
		case n == 3:
			return "background:#ef5350;color:#fff"
		case n == 4:
			return "background:#c62828;color:#fff"
		default:
			return "background:#8b0000;color:#fff"
		}
	},
	"pct": func(f float64) string { return fmt.Sprintf("%.2f", f) },
	"ns":  func(f float64) string { return fmt.Sprintf("%.1f", f) },
	"qps": func(f float64) string {
		if f >= 1_000_000 {
			return fmt.Sprintf("%.2fM", f/1_000_000)
		}
		return fmt.Sprintf("%.0fK", f/1_000)
	},
	"barWidth": func(value, max float64) float64 {
		if max == 0 {
			return 0
		}
		return (value / max) * 100
	},
	"maxLatency": func(results []benchmark.LatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.GetNsOp > max {
				max = r.GetNsOp
			}
		}
		return max
	},
	"maxSetLatency": func(results []benchmark.LatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.SetNsOp > max {
				max = r.SetNsOp
			}
		}
		return max
	},
	"maxSetEvictLatency": func(results []benchmark.LatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.SetEvictNsOp > max {
				max = r.SetEvictNsOp
			}
		}
		return max
	},
	"maxIntLatency": func(results []benchmark.IntLatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.GetNsOp > max {
				max = r.GetNsOp
			}
		}
		return max
	},
	"maxIntSetLatency": func(results []benchmark.IntLatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.SetNsOp > max {
				max = r.SetNsOp
			}
		}
		return max
	},
	"maxIntSetEvictLatency": func(results []benchmark.IntLatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.SetEvictNsOp > max {
				max = r.SetEvictNsOp
			}
		}
		return max
	},
	"maxGetOrSetLatency": func(results []benchmark.GetOrSetLatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.NsOp > max {
				max = r.NsOp
			}
		}
		return max
	},
	"sortGetOrSet": func(results []benchmark.GetOrSetLatencyResult) []benchmark.GetOrSetLatencyResult {
		sorted := make([]benchmark.GetOrSetLatencyResult, len(results))
		copy(sorted, results)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].NsOp < sorted[j].NsOp
		})
		return sorted
	},
	"maxQPS": func(results []benchmark.ThroughputResult, threads int) float64 {
		max := 0.0
		for _, r := range results {
			if r.QPS[threads] > max {
				max = r.QPS[threads]
			}
		}
		return max
	},
	"maxOverhead": func(results []benchmark.MemoryResult) float64 {
		max := int64(0)
		for _, r := range results {
			if r.BytesPerItem > max {
				max = r.BytesPerItem
			}
		}
		return float64(max)
	},
	"mb": func(b uint64) string {
		return fmt.Sprintf("%.2f", float64(b)/1024/1024)
	},
	"toFloat":      func(b uint64) float64 { return float64(b) },
	"toFloatInt":   func(b int64) float64 { return float64(b) },
	"toFloat64Int": func(b int) float64 { return float64(b) },
}

var cacheColors = map[string]string{
	"multicache":    "#2E7D32",
	"otter":         "#1976D2",
	"theine":        "#D32F2F",
	"ristretto":     "#7B1FA2",
	"freecache":     "#F57C00",
	"freelru-shard": "#0288D1",
	"freelru-sync":  "#00796B",
	"tinylfu":       "#C2185B",
	"sieve":         "#5D4037",
	"s3-fifo":       "#455A64",
	"2q":            "#E64A19",
	"s4lru":         "#512DA8",
	"clock":         "#00695C",
	"lru":           "#AFB42B",
	"ttlcache":      "#0097A7",
}
