package output

import (
	"encoding/json"
	"os"
	"time"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

type jsonResults struct {
	Timestamp   string           `json:"timestamp"`
	MachineInfo MachineInfo      `json:"machineInfo"`
	HitRate     *jsonHitRateData `json:"hitRate,omitempty"`
	Latency     *jsonLatencyData `json:"latency,omitempty"`
	Throughput  *jsonThroughput  `json:"throughput,omitempty"`
	Memory      *MemoryData      `json:"memory,omitempty"`
	MedalTable  *jsonMedalTable  `json:"medalTable,omitempty"`
	Rankings    []Ranking        `json:"rankings,omitempty"`
}

type jsonMedalTable struct {
	Categories []jsonCategoryMedals `json:"categories"`
}

type jsonCategoryMedals struct {
	Name       string               `json:"name"`
	Benchmarks []jsonBenchmarkMedal `json:"benchmarks"`
	Rankings   []jsonCategoryRank   `json:"rankings"`
}

type jsonBenchmarkMedal struct {
	Name   string   `json:"name"`
	Gold   []string `json:"gold,omitempty"`
	Silver []string `json:"silver,omitempty"`
	Bronze []string `json:"bronze,omitempty"`
}

type jsonCategoryRank struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Rank   int     `json:"rank"`
	Gold   int     `json:"gold"`
	Silver int     `json:"silver"`
	Bronze int     `json:"bronze"`
}

type jsonHitRateData struct {
	Sizes        []int               `json:"sizes"`
	CDN          []jsonHitRateResult `json:"cdn,omitempty"`
	Meta         []jsonHitRateResult `json:"meta,omitempty"`
	Zipf         []jsonHitRateResult `json:"zipf,omitempty"`
	Twitter      []jsonHitRateResult `json:"twitter,omitempty"`
	Wikipedia    []jsonHitRateResult `json:"wikipedia,omitempty"`
	ThesiosBlock []jsonHitRateResult `json:"thesiosBlock,omitempty"`
	ThesiosFile  []jsonHitRateResult `json:"thesiosFile,omitempty"`
	IBMDocker    []jsonHitRateResult `json:"ibmDocker,omitempty"`
	TencentPhoto []jsonHitRateResult `json:"tencentPhoto,omitempty"`
}

type jsonHitRateResult struct {
	Rates   map[int]float64 `json:"rates"`
	Name    string          `json:"name"`
	AvgRate float64         `json:"avgRate"`
}

type jsonLatencyData struct {
	StringKeys []jsonLatencyResult  `json:"stringKeys,omitempty"`
	IntKeys    []jsonLatencyResult  `json:"intKeys,omitempty"`
	GetOrSet   []jsonGetOrSetResult `json:"getOrSet,omitempty"`
}

type jsonLatencyResult struct {
	Name           string  `json:"name"`
	GetNsOp        float64 `json:"getNsOp"`
	GetAllocs      int64   `json:"getAllocs"`
	SetNsOp        float64 `json:"setNsOp"`
	SetAllocs      int64   `json:"setAllocs"`
	SetEvictNsOp   float64 `json:"setEvictNsOp"`
	SetEvictAllocs int64   `json:"setEvictAllocs"`
	AvgNsOp        float64 `json:"avgNsOp"`
}

type jsonGetOrSetResult struct {
	Name   string  `json:"name"`
	NsOp   float64 `json:"nsOp"`
	Allocs int64   `json:"allocs"`
}

type jsonThroughput struct {
	Threads   []int                  `json:"threads"`
	StringGet []jsonThroughputResult `json:"stringGet,omitempty"`
	StringSet []jsonThroughputResult `json:"stringSet,omitempty"`
	IntGet    []jsonThroughputResult `json:"intGet,omitempty"`
	IntSet    []jsonThroughputResult `json:"intSet,omitempty"`
	GetOrSet  []jsonThroughputResult `json:"getOrSet,omitempty"`
}

type jsonThroughputResult struct {
	QPS    map[int]float64 `json:"qps"`
	Name   string          `json:"name"`
	AvgQPS float64         `json:"avgQps"`
}

// WriteJSON writes benchmark results to a JSON file.
func WriteJSON(filename string, results Results, commandLine string) error {
	jr := convertToJSON(results, commandLine)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // best-effort close

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}

func convertToJSON(results Results, commandLine string) jsonResults {
	jr := jsonResults{
		Timestamp:   time.Now().Format(time.RFC3339),
		MachineInfo: results.MachineInfo,
		Rankings:    results.Rankings,
	}
	jr.MachineInfo.CommandLine = commandLine

	if results.HitRate != nil {
		jr.HitRate = convertHitRate(results.HitRate)
	}
	if results.Latency != nil {
		jr.Latency = convertLatency(results.Latency)
	}
	if results.Throughput != nil {
		jr.Throughput = convertThroughput(results.Throughput)
	}
	if results.Memory != nil {
		jr.Memory = results.Memory
	}
	if results.MedalTable != nil {
		jr.MedalTable = convertMedalTable(results.MedalTable)
	}

	return jr
}

func convertMedalTable(mt *MedalTable) *jsonMedalTable {
	categories := make([]jsonCategoryMedals, len(mt.Categories))
	for i, cat := range mt.Categories {
		benchmarks := make([]jsonBenchmarkMedal, len(cat.Benchmarks))
		for j, b := range cat.Benchmarks {
			benchmarks[j] = jsonBenchmarkMedal(b)
		}

		rankings := make([]jsonCategoryRank, len(cat.Rankings))
		for j, r := range cat.Rankings {
			// Compute score: Gold=10, Silver=7, Bronze=5
			score := float64(r.Gold)*10 + float64(r.Silver)*7 + float64(r.Bronze)*5
			rankings[j] = jsonCategoryRank{
				Rank:   r.Rank,
				Name:   r.Name,
				Score:  score,
				Gold:   r.Gold,
				Silver: r.Silver,
				Bronze: r.Bronze,
			}
		}

		categories[i] = jsonCategoryMedals{
			Name:       cat.Name,
			Benchmarks: benchmarks,
			Rankings:   rankings,
		}
	}
	return &jsonMedalTable{Categories: categories}
}

func convertHitRate(hr *HitRateData) *jsonHitRateData {
	return &jsonHitRateData{
		Sizes:        hr.Sizes,
		CDN:          convertHitRateResults(hr.CDN, hr.Sizes),
		Meta:         convertHitRateResults(hr.Meta, hr.Sizes),
		Zipf:         convertHitRateResults(hr.Zipf, hr.Sizes),
		Twitter:      convertHitRateResults(hr.Twitter, hr.Sizes),
		Wikipedia:    convertHitRateResults(hr.Wikipedia, hr.Sizes),
		ThesiosBlock: convertHitRateResults(hr.ThesiosBlock, hr.Sizes),
		ThesiosFile:  convertHitRateResults(hr.ThesiosFile, hr.Sizes),
		IBMDocker:    convertHitRateResults(hr.IBMDocker, hr.Sizes),
		TencentPhoto: convertHitRateResults(hr.TencentPhoto, hr.Sizes),
	}
}

func convertHitRateResults(results []benchmark.HitRateResult, sizes []int) []jsonHitRateResult {
	if len(results) == 0 {
		return nil
	}
	out := make([]jsonHitRateResult, len(results))
	for i, r := range results {
		out[i] = jsonHitRateResult{
			Name:    r.Name,
			Rates:   r.Rates,
			AvgRate: AvgHitRate(r, sizes),
		}
	}
	return out
}

func convertLatency(lat *LatencyData) *jsonLatencyData {
	jl := &jsonLatencyData{}

	for _, r := range lat.Results {
		jl.StringKeys = append(jl.StringKeys, jsonLatencyResult{
			Name:           r.Name,
			GetNsOp:        r.GetNsOp,
			GetAllocs:      r.GetAllocs,
			SetNsOp:        r.SetNsOp,
			SetAllocs:      r.SetAllocs,
			SetEvictNsOp:   r.SetEvictNsOp,
			SetEvictAllocs: r.SetEvictAllocs,
			AvgNsOp:        (r.GetNsOp + r.SetNsOp) / 2,
		})
	}

	for _, r := range lat.IntResults {
		jl.IntKeys = append(jl.IntKeys, jsonLatencyResult{
			Name:           r.Name,
			GetNsOp:        r.GetNsOp,
			GetAllocs:      r.GetAllocs,
			SetNsOp:        r.SetNsOp,
			SetAllocs:      r.SetAllocs,
			SetEvictNsOp:   r.SetEvictNsOp,
			SetEvictAllocs: r.SetEvictAllocs,
			AvgNsOp:        (r.GetNsOp + r.SetNsOp) / 2,
		})
	}

	for _, r := range lat.GetOrSetResults {
		jl.GetOrSet = append(jl.GetOrSet, jsonGetOrSetResult{
			Name:   r.Name,
			NsOp:   r.NsOp,
			Allocs: r.Allocs,
		})
	}

	return jl
}

func convertThroughput(tp *ThroughputData) *jsonThroughput {
	return &jsonThroughput{
		Threads:   tp.Threads,
		StringGet: convertThroughputResults(tp.StringGetResults),
		StringSet: convertThroughputResults(tp.StringSetResults),
		IntGet:    convertThroughputResults(tp.IntGetResults),
		IntSet:    convertThroughputResults(tp.IntSetResults),
		GetOrSet:  convertThroughputResults(tp.GetOrSetResults),
	}
}

func convertThroughputResults(results []benchmark.ThroughputResult) []jsonThroughputResult {
	if len(results) == 0 {
		return nil
	}
	out := make([]jsonThroughputResult, len(results))
	for i, r := range results {
		out[i] = jsonThroughputResult{
			Name:   r.Name,
			QPS:    r.QPS,
			AvgQPS: avgQPS(r),
		}
	}
	return out
}
