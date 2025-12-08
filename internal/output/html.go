// Package output provides result formatting and export.
package output

import (
	"fmt"
	"html/template"
	"os"
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
	Results    []benchmark.ThroughputResult
	IntResults []benchmark.ThroughputResult
	Threads    []int
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
func WriteHTML(filename string, results Results) error {
	results.Timestamp = time.Now().Format("2006-01-02 15:04:05")

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return htmlTemplate.Execute(f, results)
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
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
		colors := []string{
			"#4CAF50", "#2196F3", "#FF9800", "#9C27B0", "#E91E63",
			"#00BCD4", "#795548", "#607D8B", "#F44336", "#3F51B5",
			"#8BC34A", "#CDDC39", "#FFC107", "#FF5722", "#009688",
		}
		var datasets []string
		for i, r := range results {
			color := colors[i%len(colors)]
			var data []string
			for _, size := range sizes {
				data = append(data, fmt.Sprintf("%.2f", r.Rates[size]))
			}
			datasets = append(datasets, fmt.Sprintf(
				`{label:"%s",data:[%s],borderColor:"%s",backgroundColor:"%s",tension:0.1,fill:false}`,
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
		colors := []string{
			"#4CAF50", "#2196F3", "#FF9800", "#9C27B0", "#E91E63",
			"#00BCD4", "#795548", "#607D8B", "#F44336", "#3F51B5",
			"#8BC34A", "#CDDC39", "#FFC107", "#FF5722", "#009688",
		}
		var datasets []string
		for i, r := range results {
			color := colors[i%len(colors)]
			var data []string
			for _, t := range threads {
				data = append(data, fmt.Sprintf("%.0f", r.QPS[t]))
			}
			datasets = append(datasets, fmt.Sprintf(
				`{label:"%s",data:[%s],borderColor:"%s",backgroundColor:"%s",tension:0.1,fill:false}`,
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
	"mb": func(b uint64) string {
		return fmt.Sprintf("%.2f", float64(b)/1024/1024)
	},
	"toFloat": func(b uint64) float64 {
		return float64(b)
	},
}).Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Go Cache Benchmark Results</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
            line-height: 1.6;
        }
        h1 {
            color: #2c3e50;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 15px;
            margin-bottom: 10px;
            font-size: 2.5em;
        }
        h2 {
            color: #34495e;
            margin-top: 50px;
            padding-bottom: 10px;
            border-bottom: 2px solid #3498db;
            font-size: 1.8em;
        }
        h3 {
            color: #555;
            margin-top: 35px;
            font-size: 1.3em;
        }
        h4 {
            color: #666;
            margin-top: 25px;
            margin-bottom: 15px;
            font-size: 1.1em;
        }
        .timestamp { color: #7f8c8d; font-size: 0.95em; margin-bottom: 30px; }
        .podium-container {
            display: flex;
            justify-content: center;
            align-items: flex-end;
            gap: 20px;
            margin: 40px 0 60px 0;
            padding: 30px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        .podium-item {
            text-align: center;
            background: white;
            border-radius: 12px;
            padding: 25px 30px;
            min-width: 200px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.1);
            transition: transform 0.3s ease;
        }
        .podium-item:hover {
            transform: translateY(-5px);
        }
        .podium-item.first {
            order: 2;
            padding: 35px 40px;
            background: linear-gradient(135deg, #FFD700 0%, #FFA500 100%);
            color: #333;
        }
        .podium-item.second {
            order: 1;
            padding: 25px 30px;
            background: linear-gradient(135deg, #C0C0C0 0%, #A8A8A8 100%);
            color: #333;
        }
        .podium-item.third {
            order: 3;
            padding: 20px 25px;
            background: linear-gradient(135deg, #CD7F32 0%, #B87333 100%);
            color: white;
        }
        .medal {
            font-size: 4em;
            margin-bottom: 10px;
            display: block;
            animation: float 3s ease-in-out infinite;
        }
        @keyframes float {
            0%, 100% { transform: translateY(0); }
            50% { transform: translateY(-10px); }
        }
        .rank { font-size: 1.1em; font-weight: 600; opacity: 0.9; }
        .cache-name { font-size: 1.8em; font-weight: bold; margin: 10px 0; }
        .score { font-size: 1.1em; opacity: 0.95; font-weight: 500; }
        .section-header {
            background: white;
            padding: 20px 30px;
            border-radius: 10px;
            margin: 30px 0 20px 0;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
        }
        .section-header h2 {
            margin: 0;
            border: none;
        }
        .toc {
            background: white;
            border-radius: 10px;
            padding: 25px 30px;
            margin: 30px 0;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
        }
        .toc h2 {
            margin: 0 0 20px 0;
            color: #2c3e50;
            border: none;
            font-size: 1.5em;
        }
        .toc ul {
            list-style: none;
            padding: 0;
            margin: 0;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
        }
        .toc li {
            margin: 0;
        }
        .toc a {
            display: flex;
            align-items: center;
            padding: 12px 15px;
            background: #f8f9fa;
            border-radius: 6px;
            text-decoration: none;
            color: #333;
            transition: all 0.2s;
            border-left: 4px solid transparent;
        }
        .toc a:hover {
            background: #e9ecef;
            border-left-color: #4CAF50;
            transform: translateX(5px);
        }
        .toc a .icon {
            font-size: 1.5em;
            margin-right: 12px;
        }
        .collapsible-section {
            border: 2px solid #e0e0e0;
            border-radius: 10px;
            margin: 30px 0;
            overflow: hidden;
            background: white;
        }
        .section-toggle {
            cursor: pointer;
            padding: 20px 30px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            display: flex;
            align-items: center;
            justify-content: space-between;
            user-select: none;
            transition: background 0.3s;
        }
        .section-toggle:hover {
            background: linear-gradient(135deg, #5568d3 0%, #6a3f8f 100%);
        }
        .section-toggle h2 {
            margin: 0;
            color: white;
            border: none;
            font-size: 1.8em;
        }
        .section-toggle .toggle-icon {
            font-size: 1.5em;
            transition: transform 0.3s;
        }
        .section-toggle.collapsed .toggle-icon {
            transform: rotate(-90deg);
        }
        .section-content {
            padding: 30px;
            max-height: 10000px;
            transition: max-height 0.3s ease, padding 0.3s ease;
            overflow: hidden;
        }
        .section-content.collapsed {
            max-height: 0;
            padding: 0 30px;
        }
        .suite-description {
            margin: 0 0 20px 0;
            padding: 15px 20px;
            background: #f8f9fa;
            border-left: 4px solid #4CAF50;
            border-radius: 4px;
            color: #555;
        }
        .chart-container {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .bar-row {
            display: flex;
            align-items: center;
            margin: 8px 0;
            height: 28px;
        }
        .bar-label {
            width: 120px;
            font-size: 13px;
            font-weight: 500;
            color: #333;
        }
        .bar-container {
            flex: 1;
            height: 22px;
            background: #e0e0e0;
            border-radius: 4px;
            overflow: hidden;
        }
        .bar {
            height: 100%;
            background: linear-gradient(90deg, #4CAF50, #8BC34A);
            border-radius: 4px;
            transition: width 0.3s ease;
        }
        .bar-value {
            width: 80px;
            text-align: right;
            font-size: 13px;
            font-weight: 500;
            color: #555;
            padding-left: 10px;
        }
        .bar.latency { background: linear-gradient(90deg, #2196F3, #03A9F4); }
        .bar.memory { background: linear-gradient(90deg, #9C27B0, #E91E63); }
        .bar.throughput { background: linear-gradient(90deg, #FF9800, #FFC107); }
        table {
            width: 100%;
            border-collapse: collapse;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        th, td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #eee;
        }
        th {
            background: #4CAF50;
            color: white;
            font-weight: 500;
        }
        tr:hover { background: #f9f9f9; }
        .best { font-weight: bold; color: #4CAF50; }
        th.sortable {
            cursor: pointer;
            user-select: none;
            position: relative;
            padding-right: 25px;
        }
        th.sortable:hover {
            background: #45a049;
        }
        th.sortable::after {
            content: '▼';
            position: absolute;
            right: 8px;
            opacity: 0.3;
            font-size: 0.8em;
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
            height: 14px;
            background: linear-gradient(90deg, #2196F3, #03A9F4);
            border-radius: 3px;
            min-width: 2px;
        }
        .cell-bar.set { background: linear-gradient(90deg, #4CAF50, #8BC34A); }
        .cell-bar.evict { background: linear-gradient(90deg, #FF9800, #FFC107); }
        .cell-value {
            white-space: nowrap;
            min-width: 50px;
        }
        .line-chart-container {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            height: 400px;
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <h1>🚀 Go Cache Benchmark Results</h1>
    <p class="timestamp">Generated: {{.Timestamp}}</p>

{{if .Rankings}}
    <div class="section-header">
        <h2>🏆 Overall Winners</h2>
        <p style="margin: 10px 0 0 0; color: #666;">Ranked voting across all benchmarks</p>
    </div>

    <div class="podium-container">
        {{range $i, $r := .Rankings}}
        {{if lt $i 3}}
        <div class="podium-item {{if eq $i 0}}first{{else if eq $i 1}}second{{else}}third{{end}}">
            <span class="medal">{{if eq $i 0}}🥇{{else if eq $i 1}}🥈{{else}}🥉{{end}}</span>
            <div class="rank">{{if eq $i 0}}1st Place{{else if eq $i 1}}2nd Place{{else}}3rd Place{{end}}</div>
            <div class="cache-name">{{$r.Name}}</div>
            <div class="score">{{printf "%.0f" $r.Score}} points</div>
        </div>
        {{end}}
        {{end}}
    </div>
{{end}}

<div class="toc">
    <h2>📑 Table of Contents</h2>
    <ul>
        {{if .HitRate}}<li><a href="#hitrate"><span class="icon">📊</span> Hit Rate Benchmarks</a></li>{{end}}
        {{if .Latency}}<li><a href="#latency"><span class="icon">⚡</span> Latency Benchmarks</a></li>{{end}}
        {{if .Throughput}}<li><a href="#throughput"><span class="icon">🔥</span> Throughput Benchmarks</a></li>{{end}}
        {{if .Memory}}<li><a href="#memory"><span class="icon">💾</span> Memory Benchmarks</a></li>{{end}}
    </ul>
</div>

{{if .HitRate}}
<div class="collapsible-section" id="hitrate">
    <div class="section-toggle" onclick="toggleSection(this)">
        <h2>📊 Hit Rate Benchmarks</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Higher hit rates mean better cache effectiveness. Tests measure how often requested items are found in the cache across different cache sizes.</p>

    {{$sizes := .HitRate.Sizes}}
    {{$maxSize := index $sizes (len $sizes | add -1)}}

    {{if .HitRate.CDN}}
    <h3>CDN Production Trace</h3>
    <p style="color: #666; margin: 5px 0 15px 0;">2M operations, ~768K unique keys</p>
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
    <p style="color: #666; margin: 5px 0 15px 0;">5M operations</p>
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
    <p style="color: #666; margin: 5px 0 15px 0;">alpha=0.8, 2M operations, 100K keyspace</p>
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
    <p style="color: #666; margin: 5px 0 15px 0;">2M operations, cluster001+cluster052</p>
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
    <p style="color: #666; margin: 5px 0 15px 0;">2M operations, upload.wikimedia.org</p>
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
        <h2>⚡ Latency Benchmarks (Single-Threaded)</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Lower latency means faster operations. Measures the time (in nanoseconds) for individual Get, Set, and SetEvict operations in a single-threaded environment.</p>

    {{$sortedResults := sortByGetLatency .Latency.Results}}
    {{$maxGet := maxLatency $sortedResults}}
    {{$maxSet := maxSetLatency $sortedResults}}
    {{$maxEvict := maxSetEvictLatency $sortedResults}}

    <h3>String Keys</h3>

    <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; margin: 20px 0;">
        <div>
            <h4 style="text-align: center; color: #2196F3; margin-top: 0;">Get Latency (ns/op)</h4>
            <div class="chart-container" style="padding: 15px;">
                {{range $sortedResults}}
                <div class="bar-row">
                    <span class="bar-label">{{.Name}}</span>
                    <div class="bar-container">
                        <div class="bar latency" style="width: {{barWidth .GetNsOp $maxGet}}%"></div>
                    </div>
                    <span class="bar-value">{{ns .GetNsOp}}</span>
                </div>
                {{end}}
            </div>
        </div>

        <div>
            <h4 style="text-align: center; color: #4CAF50; margin-top: 0;">Set Latency (ns/op)</h4>
            <div class="chart-container" style="padding: 15px;">
                {{range $sortedResults}}
                <div class="bar-row">
                    <span class="bar-label">{{.Name}}</span>
                    <div class="bar-container">
                        <div class="bar set" style="width: {{barWidth .SetNsOp $maxSet}}%"></div>
                    </div>
                    <span class="bar-value">{{ns .SetNsOp}}</span>
                </div>
                {{end}}
            </div>
        </div>

        <div>
            <h4 style="text-align: center; color: #FF9800; margin-top: 0;">SetEvict Latency (ns/op)</h4>
            <div class="chart-container" style="padding: 15px;">
                {{range $sortedResults}}
                <div class="bar-row">
                    <span class="bar-label">{{.Name}}</span>
                    <div class="bar-container">
                        <div class="bar evict" style="width: {{barWidth .SetEvictNsOp $maxEvict}}%"></div>
                    </div>
                    <span class="bar-value">{{ns .SetEvictNsOp}}</span>
                </div>
                {{end}}
            </div>
        </div>
    </div>

    <h4>Full Results Table</h4>
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

    <h3 style="margin-top: 40px;">Int Keys</h3>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin: 20px 0;">
        <div>
            <h4 style="text-align: center; color: #2196F3; margin-top: 0;">Get Latency (ns/op)</h4>
            <div class="chart-container" style="padding: 15px;">
                {{range $sortedIntResults}}
                <div class="bar-row">
                    <span class="bar-label">{{.Name}}</span>
                    <div class="bar-container">
                        <div class="bar latency" style="width: {{barWidth .GetNsOp $maxIntGet}}%"></div>
                    </div>
                    <span class="bar-value">{{ns .GetNsOp}}</span>
                </div>
                {{end}}
            </div>
        </div>

        <div>
            <h4 style="text-align: center; color: #4CAF50; margin-top: 0;">Set Latency (ns/op)</h4>
            <div class="chart-container" style="padding: 15px;">
                {{range $sortedIntResults}}
                <div class="bar-row">
                    <span class="bar-label">{{.Name}}</span>
                    <div class="bar-container">
                        <div class="bar set" style="width: {{barWidth .SetNsOp $maxIntSet}}%"></div>
                    </div>
                    <span class="bar-value">{{ns .SetNsOp}}</span>
                </div>
                {{end}}
            </div>
        </div>
    </div>

    <h4>Full Results Table</h4>
    <table>
        <thead>
        <tr>
            <th class="sortable">Cache</th>
            <th class="sortable">Get (ns)</th>
            <th class="sortable">Get allocs</th>
            <th class="sortable">Set (ns)</th>
            <th class="sortable">Set allocs</th>
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
            <td style="font-weight:bold;">{{ns (avgIntLatency .)}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}
    </div>
</div>
{{end}}

{{if .Throughput}}
<div class="collapsible-section" id="throughput">
    <div class="section-toggle" onclick="toggleSection(this)">
        <h2>🔥 Throughput Benchmarks (Multi-Threaded)</h2>
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
        <h2>💾 Memory Overhead Benchmarks</h2>
        <span class="toggle-icon">▼</span>
    </div>
    <div class="section-content">
        <p class="suite-description">Capacity: {{.Memory.Capacity}} items, Value size: {{.Memory.ValSize}} bytes. Measured in isolated processes. Overhead compared to baseline <code>map[string][]byte</code>. Lower is better.</p>

    <h3>Memory Usage (MB)</h3>
    <div class="chart-container">
        {{$results := .Memory.Results}}
        {{$max := maxMemory $results}}
        {{range $results}}
        <div class="bar-row">
            <span class="bar-label">{{.Name}}</span>
            <div class="bar-container">
                <div class="bar memory" style="width: {{barWidth (toFloat .Bytes) $max}}%"></div>
            </div>
            <span class="bar-value">{{mb .Bytes}} MB</span>
        </div>
        {{end}}
    </div>

    <h3>Full Memory Table</h3>
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
        {{range .Memory.Results}}
        <tr>
            <td>{{.Name}}</td>
            <td>{{.Items}}</td>
            <td>{{mb .Bytes}}</td>
            <td>{{.BytesPerItem}}</td>
        </tr>
        {{end}}
        </tbody>
    </table>
    </div>
</div>
{{end}}

<script>
function createLineChart(canvasId, labels, datasets, yLabel) {
    const ctx = document.getElementById(canvasId);
    if (!ctx) return;
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
                y: {
                    title: { display: true, text: yLabel },
                    beginAtZero: false
                },
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

    // Smooth scroll for TOC links
    document.querySelectorAll('.toc a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const target = document.querySelector(link.getAttribute('href'));
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
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

{{if .HitRate}}
{{$sizes := .HitRate.Sizes}}
{{if .HitRate.CDN}}
createLineChart('cdnChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.CDN $sizes}}, 'Hit Rate (%)');
{{end}}
{{if .HitRate.Meta}}
createLineChart('metaChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Meta $sizes}}, 'Hit Rate (%)');
{{end}}
{{if .HitRate.Zipf}}
createLineChart('zipfChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Zipf $sizes}}, 'Hit Rate (%)');
{{end}}
{{if .HitRate.Twitter}}
createLineChart('twitterChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Twitter $sizes}}, 'Hit Rate (%)');
{{end}}
{{if .HitRate.Wikipedia}}
createLineChart('wikipediaChart', {{sizeLabels $sizes}}, {{hitRateDatasets .HitRate.Wikipedia $sizes}}, 'Hit Rate (%)');
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
</script>
</body>
</html>
`))

