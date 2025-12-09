// Package output provides result formatting and export.
package output

import (
	"fmt"
	"html/template"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

// Results holds all benchmark results for HTML output.
type Results struct {
	Timestamp   string
	HitRate     *HitRateData
	Latency     *LatencyData
	Throughput  *ThroughputData
	Memory      *MemoryData
	Rankings    []Ranking
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
	Rank  int
	Name  string
	Score float64
}

// MemoryData holds memory benchmark data.
type MemoryData struct {
	Results  []benchmark.MemoryResult
	Capacity int
	ValSize  int
}

// HitRateData holds hit rate benchmark data.
type HitRateData struct {
	CDN       []benchmark.HitRateResult
	Meta      []benchmark.HitRateResult
	Zipf      []benchmark.HitRateResult
	Twitter   []benchmark.HitRateResult
	Wikipedia []benchmark.HitRateResult
	Sizes     []int
}

// LatencyData holds latency benchmark data.
type LatencyData struct {
	Results    []benchmark.LatencyResult
	IntResults []benchmark.IntLatencyResult
}

// ThroughputData holds throughput benchmark data.
type ThroughputData struct {
	Results          []benchmark.ThroughputResult
	IntResults       []benchmark.ThroughputResult
	GetOrSetResults  []benchmark.ThroughputResult
	IntGetOrSetResults []benchmark.ThroughputResult
	Threads          []int
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}

// WriteHTML writes benchmark results to an HTML file with bar charts.
func WriteHTML(filename string, results Results, commandLine string) error {
	results.Timestamp = time.Now().Format("2006-01-02 15:04:05 MST")
	results.MachineInfo = GetMachineInfo(commandLine)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return htmlTemplate.Execute(f, results)
}

// GetMachineInfo collects information about the benchmark environment.
func GetMachineInfo(commandLine string) MachineInfo {
	return MachineInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		NumCPU:      runtime.NumCPU(),
		GoVersion:   runtime.Version(),
		CommandLine: commandLine,
	}
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"mul": func(a, b float64) float64 { return a * b },
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
	"divK":    func(n int) int { return n / 1024 },
	"toJSON":  func(v any) template.JS { return template.JS(fmt.Sprintf("%v", v)) },
	"sizeLabels": func(sizes []int) template.JS {
		labels := make([]string, len(sizes))
		for i, s := range sizes {
			labels[i] = fmt.Sprintf("\"%dK\"", s/1024)
		}
		return template.JS("[" + joinStrings(labels, ",") + "]")
	},
	"hitRateDatasets": func(results []benchmark.HitRateResult, sizes []int) template.JS {
		cacheColors := map[string]string{
			"sfcache":        "#2E7D32",
			"otter":          "#1976D2",
			"theine":         "#D32F2F",
			"ristretto":      "#7B1FA2",
			"freecache":      "#F57C00",
			"freelru-shard":  "#0288D1",
			"freelru-sync":   "#00796B",
			"tinylfu":        "#C2185B",
			"sieve":          "#5D4037",
			"s3-fifo":        "#455A64",
			"2q":             "#E64A19",
			"s4lru":          "#512DA8",
			"clock":          "#00695C",
			"lru":            "#AFB42B",
			"ttlcache":       "#0097A7",
		}
		fallbackColors := []string{
			"#388E3C", "#1E88E5", "#E53935", "#8E24AA", "#FB8C00",
		}
		var datasets []string
		for i, r := range results {
			color, ok := cacheColors[r.Name]
			if !ok {
				color = fallbackColors[i%len(fallbackColors)]
			}
			var data []string
			for _, size := range sizes {
				data = append(data, fmt.Sprintf("%.2f", r.Rates[size]))
			}
			datasets = append(datasets, fmt.Sprintf(
				`{label:"%s",data:[%s],borderColor:"%s",backgroundColor:"%s",tension:0.1,fill:false,pointRadius:3,pointHoverRadius:5}`,
				r.Name, joinStrings(data, ","), color, color,
			))
		}
		return template.JS("[" + joinStrings(datasets, ",") + "]")
	},
	"threadLabels": func(threads []int) template.JS {
		labels := make([]string, len(threads))
		for i, t := range threads {
			labels[i] = fmt.Sprintf("\"%dT\"", t)
		}
		return template.JS("[" + joinStrings(labels, ",") + "]")
	},
	"throughputDatasets": func(results []benchmark.ThroughputResult, threads []int) template.JS {
		cacheColors := map[string]string{
			"sfcache":        "#2E7D32",
			"otter":          "#1976D2",
			"theine":         "#D32F2F",
			"ristretto":      "#7B1FA2",
			"freecache":      "#F57C00",
			"freelru-shard":  "#0288D1",
			"freelru-sync":   "#00796B",
			"tinylfu":        "#C2185B",
			"sieve":          "#5D4037",
			"s3-fifo":        "#455A64",
			"2q":             "#E64A19",
			"s4lru":          "#512DA8",
			"clock":          "#00695C",
			"lru":            "#AFB42B",
			"ttlcache":       "#0097A7",
		}
		fallbackColors := []string{
			"#388E3C", "#1E88E5", "#E53935", "#8E24AA", "#FB8C00",
		}
		var datasets []string
		for i, r := range results {
			color, ok := cacheColors[r.Name]
			if !ok {
				color = fallbackColors[i%len(fallbackColors)]
			}
			var data []string
			for _, t := range threads {
				data = append(data, fmt.Sprintf("%.0f", r.QPS[t]))
			}
			datasets = append(datasets, fmt.Sprintf(
				`{label:"%s",data:[%s],borderColor:"%s",backgroundColor:"%s",tension:0.1,fill:false,pointRadius:3,pointHoverRadius:5}`,
				r.Name, joinStrings(data, ","), color, color,
			))
		}
		return template.JS("[" + joinStrings(datasets, ",") + "]")
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
	"maxHitRate": func(results []benchmark.HitRateResult, size int) float64 {
		max := 0.0
		for _, r := range results {
			if r.Rates[size] > max {
				max = r.Rates[size]
			}
		}
		return max
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
	"minLatency": func(results []benchmark.LatencyResult) float64 {
		min := results[0].GetNsOp
		for _, r := range results {
			if r.GetNsOp < min {
				min = r.GetNsOp
			}
		}
		return min
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
	"maxGetOrSetLatency": func(results []benchmark.LatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.HasGetOrSet && r.GetOrSetNsOp > max {
				max = r.GetOrSetNsOp
			}
		}
		return max
	},
	"maxIntGetOrSetLatency": func(results []benchmark.IntLatencyResult) float64 {
		max := 0.0
		for _, r := range results {
			if r.HasGetOrSet && r.GetOrSetNsOp > max {
				max = r.GetOrSetNsOp
			}
		}
		return max
	},
	"filterGetOrSet": func(results []benchmark.LatencyResult) []benchmark.LatencyResult {
		filtered := make([]benchmark.LatencyResult, 0)
		for _, r := range results {
			if r.HasGetOrSet {
				filtered = append(filtered, r)
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].GetOrSetNsOp < filtered[j].GetOrSetNsOp
		})
		return filtered
	},
	"filterIntGetOrSet": func(results []benchmark.IntLatencyResult) []benchmark.IntLatencyResult {
		filtered := make([]benchmark.IntLatencyResult, 0)
		for _, r := range results {
			if r.HasGetOrSet {
				filtered = append(filtered, r)
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].GetOrSetNsOp < filtered[j].GetOrSetNsOp
		})
		return filtered
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
	"maxMemory": func(results []benchmark.MemoryResult) float64 {
		var max uint64
		for _, r := range results {
			if r.Bytes > max {
				max = r.Bytes
			}
		}
		return float64(max)
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
	"toFloat": func(b uint64) float64 {
		return float64(b)
	},
	"toFloatInt": func(b int64) float64 {
		return float64(b)
	},
}).Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>gocachemark - Go Cache Benchmark Results</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 40px 20px;
            background: white;
            line-height: 1.6;
            color: #333;
        }
        @media print {
            body { padding: 20px; }
            .podium-container { page-break-inside: avoid; }
            .chart-container { page-break-inside: avoid; }
        }
        h1 {
            color: #000;
            border-bottom: 2px solid #000;
            padding-bottom: 10px;
            margin-bottom: 10px;
            font-size: 2em;
            font-weight: 600;
        }
        h2 {
            color: #000;
            margin-top: 50px;
            padding-bottom: 8px;
            border-bottom: 1px solid #ccc;
            font-size: 1.5em;
            font-weight: 600;
        }
        h3 {
            color: #333;
            margin-top: 30px;
            font-size: 1.2em;
            font-weight: 600;
        }
        h4 {
            color: #333;
            margin-top: 20px;
            margin-bottom: 10px;
            font-size: 1em;
            font-weight: 600;
        }
        .timestamp {
            color: #666;
            font-size: 0.95em;
            margin-bottom: 20px;
        }
        .timestamp a {
            color: #0066cc;
            text-decoration: none;
        }
        .timestamp a:hover {
            text-decoration: underline;
        }
        .benchmark-info-top {
            padding: 15px;
            margin-bottom: 30px;
            border: 1px solid #ddd;
            background: #fafafa;
            font-size: 0.9em;
        }
        .benchmark-info-grid {
            display: grid;
            grid-template-columns: repeat(3, auto);
            gap: 20px;
            margin-bottom: 10px;
        }
        .benchmark-info-top .info-item {
            display: flex;
            gap: 8px;
            white-space: nowrap;
        }
        .benchmark-info-command {
            display: flex;
            gap: 8px;
            padding-top: 10px;
            border-top: 1px solid #ddd;
        }
        .benchmark-info-top .info-label {
            font-weight: 600;
            color: #000;
        }
        .benchmark-info-top .info-value {
            color: #555;
            font-family: "SF Mono", Monaco, "Courier New", monospace;
        }
        .benchmark-info-command .info-value {
            overflow-x: auto;
            white-space: nowrap;
        }
        .podium-container {
            display: flex;
            justify-content: center;
            align-items: flex-end;
            gap: 20px;
            margin: 40px 0;
            padding: 30px;
            border: 1px solid #ddd;
            height: 280px;
        }
        .podium-item {
            text-align: center;
            border: 2px solid #000;
            padding: 20px 30px;
            min-width: 180px;
            display: flex;
            flex-direction: column;
            justify-content: flex-end;
        }
        .podium-item.first {
            order: 2;
            background: #FFD700;
            border-color: #DAA520;
            border-width: 3px;
        }
        .podium-item.second {
            order: 1;
            background: #C0C0C0;
            border-color: #A8A8A8;
        }
        .podium-item.third {
            order: 3;
            background: #CD7F32;
            border-color: #B87333;
            color: #000;
        }
        .medal {
            font-size: 2.5em;
            margin-bottom: 10px;
            display: block;
        }
        .rank {
            font-size: 0.9em;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .cache-name {
            font-size: 1.5em;
            font-weight: 700;
            margin: 10px 0;
            font-family: "SF Mono", Monaco, "Courier New", monospace;
        }
        .score {
            font-size: 1em;
            font-weight: 500;
            color: #666;
        }
        .section-header {
            padding: 15px 0;
            margin: 40px 0 20px 0;
            border-bottom: 1px solid #ddd;
        }
        .section-header h2 {
            margin: 0;
            border: none;
        }
        .section-header p {
            margin: 10px 0 0 0;
            color: #666;
            font-size: 0.9em;
        }
        .toc {
            padding: 20px 0;
            margin: 30px 0;
            border-top: 1px solid #ddd;
            border-bottom: 1px solid #ddd;
        }
        .toc h2 {
            margin: 0 0 15px 0;
            color: #000;
            border: none;
            font-size: 1.2em;
        }
        .toc ul {
            list-style: disc;
            padding-left: 20px;
            margin: 0;
        }
        .toc li {
            margin: 8px 0;
        }
        .toc a {
            text-decoration: none;
            color: #0066cc;
        }
        .toc a:hover {
            text-decoration: underline;
        }
        .toc a .icon {
            margin-right: 8px;
        }
        .collapsible-section {
            border: 1px solid #ddd;
            margin: 30px 0;
            background: white;
        }
        .section-toggle {
            cursor: pointer;
            padding: 15px 20px;
            background: #f5f5f5;
            border-bottom: 1px solid #ddd;
            color: #000;
            display: flex;
            align-items: center;
            justify-content: space-between;
            user-select: none;
        }
        .section-toggle h2 {
            margin: 0;
            color: #000;
            border: none;
            font-size: 1.5em;
            font-weight: 600;
        }
        .section-toggle .toggle-icon {
            font-size: 1.5em;
        }
        .section-toggle.collapsed .toggle-icon {
            transform: rotate(-90deg);
        }
        .section-content {
            padding: 30px;
        }
        .section-content.collapsed {
            display: none;
        }
        .suite-description {
            margin: 0 0 20px 0;
            padding: 10px 0 10px 15px;
            border-left: 3px solid #666;
            color: #555;
        }
        .chart-container {
            padding: 20px;
            margin: 30px 0;
            border: 1px solid #ddd;
        }
        footer {
            margin-top: 60px;
            padding: 20px 0;
            border-top: 1px solid #ddd;
            color: #999;
            font-size: 0.85em;
            text-align: center;
        }
        footer a {
            color: #0066cc;
            text-decoration: none;
        }
        footer a:hover {
            text-decoration: underline;
        }
        .bar-row {
            display: flex;
            align-items: center;
            margin: 6px 0;
            height: 24px;
        }
        .bar-label {
            width: 120px;
            font-size: 13px;
            font-weight: 500;
            color: #000;
            font-family: "SF Mono", Monaco, "Courier New", monospace;
        }
        .bar-container {
            flex: 1;
            height: 18px;
            background: #f0f0f0;
            border: 1px solid #ccc;
            overflow: hidden;
        }
        .bar {
            height: 100%;
            background: #2E7D32;
        }
        .bar-value {
            width: 90px;
            text-align: right;
            font-size: 13px;
            font-weight: 500;
            color: #333;
            padding-left: 10px;
            font-family: "SF Mono", Monaco, "Courier New", monospace;
        }
        .bar.latency { background: #2E7D32; }
        .bar.memory { background: #2E7D32; }
        .bar.throughput { background: #2E7D32; }
        table {
            width: 100%;
            border-collapse: collapse;
            background: white;
            border: 1px solid #ddd;
            margin: 20px 0;
        }
        th, td {
            padding: 8px 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background: #f5f5f5;
            color: #000;
            font-weight: 600;
            border-bottom: 2px solid #000;
        }
        td {
            font-family: "SF Mono", Monaco, "Courier New", monospace;
            font-size: 0.9em;
        }
        td:first-child {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            font-weight: 500;
        }
        .best { font-weight: bold; }
        th.sortable {
            cursor: pointer;
            user-select: none;
            position: relative;
            padding-right: 25px;
        }
        th.sortable:hover {
            background: #e5e5e5;
        }
        th.sortable::after {
            content: '▼';
            position: absolute;
            right: 8px;
            opacity: 0.3;
            font-size: 0.7em;
        }
        th.sortable.sorted-asc::after {
            content: '▲';
            opacity: 1;
        }
        th.sortable.sorted-desc::after {
            content: '▼';
            opacity: 1;
        }
        .cell-bar-container {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .cell-bar {
            height: 12px;
            background: #2E7D32;
            min-width: 2px;
        }
        .cell-bar.set { background: #388E3C; }
        .cell-bar.evict { background: #43A047; }
        .cell-value {
            white-space: nowrap;
            min-width: 50px;
        }
        .line-chart-container {
            padding: 20px;
            margin: 30px 0;
            border: 1px solid #ddd;
            height: 400px;
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <h1>gocachemark</h1>
    <p class="timestamp">
        Benchmark report for Go cache implementations.
        <a href="https://github.com/tstromberg/gocachemark" target="_blank">View on GitHub →</a>
    </p>

    <div class="benchmark-info-top">
        <div class="benchmark-info-grid">
            <div class="info-item">
                <span class="info-label">Generated:</span>
                <span class="info-value">{{.Timestamp}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Machine:</span>
                <span class="info-value">{{.MachineInfo.OS}}/{{.MachineInfo.Arch}} ({{.MachineInfo.NumCPU}} CPUs)</span>
            </div>
            <div class="info-item">
                <span class="info-label">Go Version:</span>
                <span class="info-value">{{.MachineInfo.GoVersion}}</span>
            </div>
        </div>
        <div class="benchmark-info-command info-item">
            <span class="info-label">Command:</span>
            <span class="info-value">{{.MachineInfo.CommandLine}}</span>
        </div>
    </div>

{{if .Rankings}}
    <div class="section-header">
        <h2>Overall Winners</h2>
        <p>Ranked voting across all benchmarks</p>
    </div>

    <div class="podium-container">
        {{$first := index .Rankings 0}}
        {{$maxScore := $first.Score}}
        {{range $i, $r := .Rankings}}
        {{if lt $i 3}}
        {{$heightPx := 200}}
        {{if ne $maxScore 0.0}}
            {{$heightPx = printf "%.0f" (mul (div $r.Score $maxScore) 200.0)}}
        {{end}}
        <div class="podium-item {{if eq $i 0}}first{{else if eq $i 1}}second{{else}}third{{end}}" style="height: {{$heightPx}}px;">
            <div class="rank">{{if eq $i 0}}1st Place{{else if eq $i 1}}2nd Place{{else}}3rd Place{{end}}</div>
            <div class="cache-name">{{$r.Name}}</div>
            <div class="score">{{printf "%.0f" $r.Score}} points</div>
        </div>
        {{end}}
        {{end}}
    </div>
{{end}}

<div class="toc">
    <h2>Table of Contents</h2>
    <ul>
        {{if .HitRate}}<li><a href="#hitrate">Hit Rate Benchmarks</a></li>{{end}}
        {{if .Latency}}<li><a href="#latency">Latency Benchmarks</a></li>{{end}}
        {{if .Throughput}}<li><a href="#throughput">Throughput Benchmarks</a></li>{{end}}
        {{if .Memory}}<li><a href="#memory">Memory Benchmarks</a></li>{{end}}
    </ul>
</div>

{{if .HitRate}}
<div class="collapsible-section" id="hitrate">
    <div class="section-toggle" onclick="toggleSection(this)">
        <h2>Hit Rate Benchmarks</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Higher hit rates mean better cache effectiveness. Tests measure how often requested items are found in the cache across different cache sizes.</p>

    {{$sizes := .HitRate.Sizes}}
    {{$maxSize := index $sizes (len $sizes | add -1)}}

    {{if .HitRate.CDN}}
    <h3>CDN Production Trace</h3>
    <p class="suite-description">2M operations, ~768K unique keys</p>
    <div class="line-chart-container">
        <canvas id="cdnChart"></canvas>
    </div>
    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $sizes}}<th class="sortable">{{divK .}}K</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByHitRate .HitRate.CDN $sizes}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $sizes}}<td>{{pct (index $r.Rates .)}}%</td>{{end}}
            <td style="font-weight:bold;">{{pct (avgHitRate $r $sizes)}}%</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .HitRate.Meta}}
    <h3 style="margin-top: 50px;">Meta KVCache Production Trace</h3>
    <p class="suite-description">5M operations</p>
    <div class="line-chart-container">
        <canvas id="metaChart"></canvas>
    </div>
    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $sizes}}<th class="sortable">{{divK .}}K</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByHitRate .HitRate.Meta $sizes}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $sizes}}<td>{{pct (index $r.Rates .)}}%</td>{{end}}
            <td style="font-weight:bold;">{{pct (avgHitRate $r $sizes)}}%</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .HitRate.Zipf}}
    <h3 style="margin-top: 50px;">Zipf Synthetic Trace</h3>
    <p class="suite-description">alpha=0.8, 2M operations, 100K keyspace</p>
    <div class="line-chart-container">
        <canvas id="zipfChart"></canvas>
    </div>
    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $sizes}}<th class="sortable">{{divK .}}K</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByHitRate .HitRate.Zipf $sizes}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $sizes}}<td>{{pct (index $r.Rates .)}}%</td>{{end}}
            <td style="font-weight:bold;">{{pct (avgHitRate $r $sizes)}}%</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .HitRate.Twitter}}
    <h3 style="margin-top: 50px;">Twitter Production Cache Trace</h3>
    <p class="suite-description">2M operations, cluster001+cluster052</p>
    <div class="line-chart-container">
        <canvas id="twitterChart"></canvas>
    </div>
    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $sizes}}<th class="sortable">{{divK .}}K</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByHitRate .HitRate.Twitter $sizes}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $sizes}}<td>{{pct (index $r.Rates .)}}%</td>{{end}}
            <td style="font-weight:bold;">{{pct (avgHitRate $r $sizes)}}%</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .HitRate.Wikipedia}}
    <h3 style="margin-top: 50px;">Wikipedia CDN Upload Trace</h3>
    <p class="suite-description">2M operations, upload.wikimedia.org</p>
    <div class="line-chart-container">
        <canvas id="wikipediaChart"></canvas>
    </div>
    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $sizes}}<th class="sortable">{{divK .}}K</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByHitRate .HitRate.Wikipedia $sizes}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $sizes}}<td>{{pct (index $r.Rates .)}}%</td>{{end}}
            <td style="font-weight:bold;">{{pct (avgHitRate $r $sizes)}}%</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}
    </div>
</div>
{{end}}

{{if .Latency}}
<div class="collapsible-section" id="latency">
    <div class="section-toggle" onclick="toggleSection(this)">
        <h2>Latency Benchmarks (Single-Threaded)</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Lower latency means faster operations. Measures the time (in nanoseconds) for individual Get, Set, and SetEvict operations in a single-threaded environment.</p>

    {{$sortedResults := sortByGetLatency .Latency.Results}}
    {{$maxGet := maxLatency $sortedResults}}
    {{$maxSet := maxSetLatency $sortedResults}}
    {{$maxEvict := maxSetEvictLatency $sortedResults}}

    <h3>String Keys</h3>

    <h4>Results</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            <th class="sortable">Get (ns)</th>
            <th class="sortable">Get allocs</th>
            <th class="sortable">Set (ns)</th>
            <th class="sortable">Set allocs</th>
            <th class="sortable">SetEvict (ns)</th>
            <th class="sortable">SetEvict allocs</th>
            <th class="sortable">Avg (ns)</th>
        </tr>
        </thead>
        <tbody>
        {{range $sortedResults}}
        <tr>
            <td>{{.Name}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar" style="width: {{barWidth .GetNsOp $maxGet}}px"></div><span class="cell-value">{{ns .GetNsOp}}</span></div></td>
            <td>{{.GetAllocs}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar set" style="width: {{barWidth .SetNsOp $maxSet}}px"></div><span class="cell-value">{{ns .SetNsOp}}</span></div></td>
            <td>{{.SetAllocs}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar evict" style="width: {{barWidth .SetEvictNsOp $maxEvict}}px"></div><span class="cell-value">{{ns .SetEvictNsOp}}</span></div></td>
            <td>{{.SetEvictAllocs}}</td>
            <td style="font-weight:bold;">{{ns (avgLatency .)}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>

    {{if .Latency.IntResults}}
    {{$sortedIntResults := sortByIntGetLatency .Latency.IntResults}}
    {{$maxIntGet := maxIntLatency $sortedIntResults}}
    {{$maxIntSet := maxIntSetLatency $sortedIntResults}}
    {{$maxIntEvict := maxIntSetEvictLatency $sortedIntResults}}

    <h3 style="margin-top: 40px;">Int Keys</h3>

    <h4>Results</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            <th class="sortable">Get (ns)</th>
            <th class="sortable">Get allocs</th>
            <th class="sortable">Set (ns)</th>
            <th class="sortable">Set allocs</th>
            <th class="sortable">SetEvict (ns)</th>
            <th class="sortable">SetEvict allocs</th>
            <th class="sortable">Avg (ns)</th>
        </tr>
        </thead>
        <tbody>
        {{range $sortedIntResults}}
        <tr>
            <td>{{.Name}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar" style="width: {{barWidth .GetNsOp $maxIntGet}}px"></div><span class="cell-value">{{ns .GetNsOp}}</span></div></td>
            <td>{{.GetAllocs}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar set" style="width: {{barWidth .SetNsOp $maxIntSet}}px"></div><span class="cell-value">{{ns .SetNsOp}}</span></div></td>
            <td>{{.SetAllocs}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar evict" style="width: {{barWidth .SetEvictNsOp $maxIntEvict}}px"></div><span class="cell-value">{{ns .SetEvictNsOp}}</span></div></td>
            <td>{{.SetEvictAllocs}}</td>
            <td style="font-weight:bold;">{{ns (avgIntLatency .)}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{$getOrSetResults := filterGetOrSet .Latency.Results}}
    {{if $getOrSetResults}}
    {{$maxGetOrSet := maxGetOrSetLatency .Latency.Results}}
    <h3 style="margin-top: 40px;">String Keys - GetOrSet</h3>
    <p class="suite-description">GetOrSet is an atomic operation that gets a value if it exists, or sets it if it doesn't. Only caches that support this operation are shown.</p>

    <h4>GetOrSet Results</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            <th class="sortable">GetOrSet (ns)</th>
            <th class="sortable">GetOrSet allocs</th>
        </tr>
        </thead>
        <tbody>
        {{range $getOrSetResults}}
        <tr>
            <td>{{.Name}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar" style="width: {{barWidth .GetOrSetNsOp $maxGetOrSet}}px; background: #2E7D32;"></div><span class="cell-value">{{ns .GetOrSetNsOp}}</span></div></td>
            <td>{{.GetOrSetAllocs}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .Latency.IntResults}}
    {{$intGetOrSetResults := filterIntGetOrSet .Latency.IntResults}}
    {{if $intGetOrSetResults}}
    {{$maxIntGetOrSet := maxIntGetOrSetLatency .Latency.IntResults}}
    <h3 style="margin-top: 40px;">Int Keys - GetOrSet</h3>
    <p class="suite-description">GetOrSet latency for integer keys. Only caches that support this operation are shown.</p>

    <h4>GetOrSet Results</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            <th class="sortable">GetOrSet (ns)</th>
            <th class="sortable">GetOrSet allocs</th>
        </tr>
        </thead>
        <tbody>
        {{range $intGetOrSetResults}}
        <tr>
            <td>{{.Name}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar" style="width: {{barWidth .GetOrSetNsOp $maxIntGetOrSet}}px; background: #2E7D32;"></div><span class="cell-value">{{ns .GetOrSetNsOp}}</span></div></td>
            <td>{{.GetOrSetAllocs}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}
    {{end}}

    </div>
</div>
{{end}}

{{if .Throughput}}
<div class="collapsible-section" id="throughput">
    <div class="section-toggle" onclick="toggleSection(this)">
        <h2>Throughput Benchmarks (Multi-Threaded)</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Higher throughput means more queries per second (QPS). Tests measure concurrent performance with a Zipf workload (75% reads, 25% writes) across different thread counts.</p>

    {{$threads := .Throughput.Threads}}
    {{$maxThreads := index $threads (len $threads | add -1)}}

    {{if .Throughput.Results}}
    <h3>String Keys</h3>
    <div class="line-chart-container">
        <canvas id="throughputChart"></canvas>
    </div>

    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $threads}}<th class="sortable">{{.}}T</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByThroughput .Throughput.Results $threads}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $threads}}<td>{{qps (index $r.QPS .)}}</td>{{end}}
            <td style="font-weight:bold;">{{qps (avgQPS $r)}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .Throughput.IntResults}}
    <h3 style="margin-top: 50px;">Int Keys</h3>
    <div class="line-chart-container">
        <canvas id="throughputIntChart"></canvas>
    </div>

    <h4>Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            {{range $threads}}<th class="sortable">{{.}}T</th>{{end}}
            <th class="sortable">Avg</th>
        </tr>
        </thead>
        <tbody>
        {{range $r := sortByThroughput .Throughput.IntResults $threads}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $threads}}<td>{{qps (index $r.QPS .)}}</td>{{end}}
            <td style="font-weight:bold;">{{qps (avgQPS $r)}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}
    </div>
</div>
{{end}}

{{if .Memory}}
<div class="collapsible-section" id="memory">
    <div class="section-toggle" onclick="toggleSection(this)">
        <h2>Memory Overhead Benchmarks</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Capacity: {{.Memory.Capacity}} items, Value size: {{.Memory.ValSize}} bytes. Measured in isolated processes. Overhead compared to baseline <code>map[string][]byte</code>. Lower is better.</p>

    {{$results := .Memory.Results}}
    {{$maxOverhead := maxOverhead $results}}
    <h3>Memory Results</h3>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            <th class="sortable">Items Stored</th>
            <th class="sortable">Memory (MB)</th>
            <th class="sortable">Overhead vs map (bytes/item)</th>
        </tr>
        </thead>
        <tbody>
        {{range $results}}
        <tr>
            <td>{{.Name}}</td>
            <td>{{.Items}}</td>
            <td>{{mb .Bytes}}</td>
            <td><div class="cell-bar-container"><div class="cell-bar" style="width: {{barWidth (toFloatInt .BytesPerItem) $maxOverhead}}px"></div><span class="cell-value">{{.BytesPerItem}}</span></div></td>
        </tr>
        {{end}}
        </tbody>
    </table>
    </div>
</div>
{{end}}

<script>
function createLineChart(canvasId, labels, datasets, yLabel, useLogY = false) {
    const ctx = document.getElementById(canvasId);
    if (!ctx) return;

    const yAxisConfig = {
        title: { display: true, text: yLabel },
        beginAtZero: false
    };

    if (useLogY) {
        yAxisConfig.type = 'logarithmic';
        yAxisConfig.min = 1;
        yAxisConfig.ticks = {
            callback: function(value, index, ticks) {
                // Only show major gridlines (powers of 10 and their midpoints)
                if (value === 1 || value === 10 || value === 100 ||
                    value === 5 || value === 50) {
                    return value;
                }
                return null;
            }
        };
    }

    new Chart(ctx, {
        type: 'line',
        data: { labels: labels, datasets: datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { position: 'right' }
            },
            scales: {
                y: yAxisConfig,
                x: {
                    title: { display: true, text: 'Cache Size' }
                }
            }
        }
    });
}

function makeSortable(table) {
    const headers = table.querySelectorAll('th.sortable');
    const tbody = table.querySelector('tbody') || table;

    headers.forEach((header, index) => {
        header.addEventListener('click', () => {
            const currentSort = header.classList.contains('sorted-desc') ? 'desc' :
                              header.classList.contains('sorted-asc') ? 'asc' : 'none';
            const newSort = currentSort === 'desc' ? 'asc' : 'desc';

            // Remove sorting from all headers
            headers.forEach(h => h.classList.remove('sorted-asc', 'sorted-desc'));

            // Add new sort class
            header.classList.add(newSort === 'asc' ? 'sorted-asc' : 'sorted-desc');

            // Get rows
            const rows = Array.from(tbody.querySelectorAll('tr'));

            // Sort rows
            rows.sort((a, b) => {
                let aVal = a.cells[index].textContent.trim();
                let bVal = b.cells[index].textContent.trim();

                // Remove % and units for numeric comparison
                aVal = aVal.replace(/%|ns|K|M|MB/g, '');
                bVal = bVal.replace(/%|ns|K|M|MB/g, '');

                // Parse as numbers if possible
                const aNum = parseFloat(aVal);
                const bNum = parseFloat(bVal);

                if (!isNaN(aNum) && !isNaN(bNum)) {
                    return newSort === 'asc' ? aNum - bNum : bNum - aNum;
                }

                // String comparison
                return newSort === 'asc' ?
                    aVal.localeCompare(bVal) :
                    bVal.localeCompare(aVal);
            });

            // Re-append rows
            rows.forEach(row => tbody.appendChild(row));
        });
    });
}

// Initialize sortable tables
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('table').forEach(table => {
        if (table.querySelector('th.sortable')) {
            makeSortable(table);

            // Trigger initial sort on Average column if it exists
            const avgHeader = Array.from(table.querySelectorAll('th.sortable'))
                .find(th => th.textContent.trim() === 'Avg' || th.textContent.trim() === 'Average');
            if (avgHeader) {
                avgHeader.click(); // Descending (best first)
            }
        }
    });

    // TOC links
    document.querySelectorAll('.toc a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const target = document.querySelector(link.getAttribute('href'));
            if (target) {
                target.scrollIntoView({ block: 'start' });
                // Expand the section if collapsed
                const content = target.querySelector('.section-content');
                const toggle = target.querySelector('.section-toggle');
                if (content && content.classList.contains('collapsed')) {
                    toggleSection(toggle);
                }
            }
        });
    });
});

function toggleSection(toggleElement) {
    const content = toggleElement.nextElementSibling;
    toggleElement.classList.toggle('collapsed');
    content.classList.toggle('collapsed');
}

function initCharts() {
{{if .HitRate}}
{{$sizes := .HitRate.Sizes}}
{{if .HitRate.CDN}}
createLineChart('cdnChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.CDN $sizes}}, 'Hit Rate (%)', false);
{{end}}
{{if .HitRate.Meta}}
createLineChart('metaChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Meta $sizes}}, 'Hit Rate (%)', false);
{{end}}
{{if .HitRate.Zipf}}
createLineChart('zipfChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Zipf $sizes}}, 'Hit Rate (%)', false);
{{end}}
{{if .HitRate.Twitter}}
createLineChart('twitterChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Twitter $sizes}}, 'Hit Rate (%)', false);
{{end}}
{{if .HitRate.Wikipedia}}
createLineChart('wikipediaChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Wikipedia $sizes}}, 'Hit Rate (%)', false);
{{end}}
{{end}}

{{if .Throughput}}
{{$threads := .Throughput.Threads}}
{{if .Throughput.Results}}
createLineChart('throughputChart', {{threadLabels $threads}}, {{throughputDatasets .Throughput.Results $threads}}, 'QPS');
{{end}}
{{if .Throughput.IntResults}}
createLineChart('throughputIntChart', {{threadLabels $threads}}, {{throughputDatasets .Throughput.IntResults $threads}}, 'QPS');
{{end}}
{{end}}
}

// Wait for Chart.js to load (needed for htmlpreview.github.io)
(function waitForChart() {
    if (typeof Chart !== 'undefined') {
        initCharts();
    } else {
        setTimeout(waitForChart, 50);
    }
})();
</script>

<footer>
    Generated by <a href="https://github.com/tstromberg/gocachemark" target="_blank">gocachemark</a>
</footer>

</body>
</html>
`))

