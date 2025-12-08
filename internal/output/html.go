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
}

// MemoryData holds memory benchmark data.
type MemoryData struct {
	Results  []benchmark.MemoryResult
	Capacity int
	ValSize  int
}

// HitRateData holds hit rate benchmark data.
type HitRateData struct {
	CDN  []benchmark.HitRateResult
	Meta []benchmark.HitRateResult
	Zipf []benchmark.HitRateResult
	Sizes []int
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
	"sortByHitRate": func(results []benchmark.HitRateResult, sizes []int) []benchmark.HitRateResult {
		sorted := make([]benchmark.HitRateResult, len(results))
		copy(sorted, results)
		maxSize := sizes[len(sizes)-1]
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Rates[maxSize] > sorted[j].Rates[maxSize]
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
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        h1 { color: #333; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 40px; }
        h3 { color: #666; margin-top: 30px; }
        .timestamp { color: #888; font-size: 0.9em; margin-bottom: 30px; }
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
    <h1>Go Cache Benchmark Results</h1>
    <p class="timestamp">Generated: {{.Timestamp}}</p>

{{if .HitRate}}
    <h2>Hit Rate Benchmarks</h2>

    {{$sizes := .HitRate.Sizes}}
    {{$maxSize := index $sizes (len $sizes | add -1)}}

    {{if .HitRate.CDN}}
    <h3>CDN Production Trace (2M ops)</h3>
    <div class="line-chart-container">
        <canvas id="cdnChart"></canvas>
    </div>
    {{end}}

    {{if .HitRate.Meta}}
    <h3>Meta KVCache Trace (5M ops)</h3>
    <div class="line-chart-container">
        <canvas id="metaChart"></canvas>
    </div>
    {{end}}

    {{if .HitRate.Zipf}}
    <h3>Zipf Synthetic (alpha=0.8, 2M ops, 100K keys)</h3>
    <div class="line-chart-container">
        <canvas id="zipfChart"></canvas>
    </div>
    {{end}}

    <h3>Full Results Table ({{divK $maxSize}}K cache)</h3>
    <table>
        <tr>
            <th>Cache</th>
            {{range $sizes}}<th>{{divK .}}K</th>{{end}}
        </tr>
        {{if .HitRate.CDN}}
        <tr><td colspan="{{len $sizes | add 1}}" style="background:#eee;font-weight:bold;">CDN Trace</td></tr>
        {{range $r := sortByHitRate .HitRate.CDN $sizes}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $sizes}}<td>{{pct (index $r.Rates .)}}%</td>{{end}}
        </tr>
        {{end}}
        {{end}}
    </table>
{{end}}

{{if .Latency}}
    <h2>Latency Benchmarks (Single-Threaded)</h2>

    <h3>String Keys - Get Latency (ns/op) - Lower is Better</h3>
    <div class="chart-container">
        {{$results := sortByGetLatency .Latency.Results}}
        {{$max := maxLatency $results}}
        {{range $results}}
        <div class="bar-row">
            <span class="bar-label">{{.Name}}</span>
            <div class="bar-container">
                <div class="bar latency" style="width: {{barWidth .GetNsOp $max}}%"></div>
            </div>
            <span class="bar-value">{{ns .GetNsOp}} ns</span>
        </div>
        {{end}}
    </div>

    <h3>String Keys - Full Table</h3>
    <table>
        <tr>
            <th>Cache</th>
            <th>Get (ns)</th>
            <th>Get allocs</th>
            <th>Set (ns)</th>
            <th>Set allocs</th>
            <th>SetEvict (ns)</th>
            <th>SetEvict allocs</th>
        </tr>
        {{range sortByGetLatency .Latency.Results}}
        <tr>
            <td>{{.Name}}</td>
            <td>{{ns .GetNsOp}}</td>
            <td>{{.GetAllocs}}</td>
            <td>{{ns .SetNsOp}}</td>
            <td>{{.SetAllocs}}</td>
            <td>{{ns .SetEvictNsOp}}</td>
            <td>{{.SetEvictAllocs}}</td>
        </tr>
        {{end}}
    </table>

    {{if .Latency.IntResults}}
    <h3>Int Keys - Get Latency (ns/op) - Lower is Better</h3>
    <div class="chart-container">
        {{$intResults := sortByIntGetLatency .Latency.IntResults}}
        {{$intMax := maxIntLatency $intResults}}
        {{range $intResults}}
        <div class="bar-row">
            <span class="bar-label">{{.Name}}</span>
            <div class="bar-container">
                <div class="bar latency" style="width: {{barWidth .GetNsOp $intMax}}%"></div>
            </div>
            <span class="bar-value">{{ns .GetNsOp}} ns</span>
        </div>
        {{end}}
    </div>

    <h3>Int Keys - Full Table</h3>
    <table>
        <tr>
            <th>Cache</th>
            <th>Get (ns)</th>
            <th>Get allocs</th>
            <th>Set (ns)</th>
            <th>Set allocs</th>
        </tr>
        {{range sortByIntGetLatency .Latency.IntResults}}
        <tr>
            <td>{{.Name}}</td>
            <td>{{ns .GetNsOp}}</td>
            <td>{{.GetAllocs}}</td>
            <td>{{ns .SetNsOp}}</td>
            <td>{{.SetAllocs}}</td>
        </tr>
        {{end}}
    </table>
    {{end}}
{{end}}

{{if .Throughput}}
    <h2>Throughput Benchmarks (Multi-Threaded)</h2>

    {{$threads := .Throughput.Threads}}
    {{$maxThreads := index $threads (len $threads | add -1)}}

    <h3>QPS by Thread Count - Higher is Better</h3>
    <div class="line-chart-container">
        <canvas id="throughputChart"></canvas>
    </div>

    <h3>Full Throughput Table</h3>
    <table>
        <tr>
            <th>Cache</th>
            {{range $threads}}<th>{{.}}T</th>{{end}}
        </tr>
        {{range $r := sortByThroughput .Throughput.Results $threads}}
        <tr>
            <td>{{$r.Name}}</td>
            {{range $threads}}<td>{{qps (index $r.QPS .)}}</td>{{end}}
        </tr>
        {{end}}
    </table>
{{end}}

{{if .Memory}}
    <h2>Memory Overhead Benchmarks</h2>
    <p>Capacity: {{.Memory.Capacity}} items, Value size: {{.Memory.ValSize}} bytes</p>
    <p>Measured in isolated processes for accuracy. Lower is better.</p>

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
        <tr>
            <th>Cache</th>
            <th>Items Stored</th>
            <th>Memory (MB)</th>
            <th>Overhead (bytes/item)</th>
        </tr>
        {{range .Memory.Results}}
        <tr>
            <td>{{.Name}}</td>
            <td>{{.Items}}</td>
            <td>{{mb .Bytes}}</td>
            <td>{{.BytesPerItem}}</td>
        </tr>
        {{end}}
    </table>
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
{{end}}

{{if .Throughput}}
{{$threads := .Throughput.Threads}}
createLineChart('throughputChart', {{threadLabels $threads}}, {{throughputDatasets .Throughput.Results $threads}}, 'QPS');
{{end}}
</script>
</body>
</html>
`))

