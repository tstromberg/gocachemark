package output

import (
	"sort"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

// Points awarded by placement: 1st=10, 2nd=7, 3rd=5, 4th=4, 5th=3, 6th=2, 7th=1.
var placementPoints = []float64{10, 7, 5, 4, 3, 2, 1}

// ComputeRankings calculates overall rankings from benchmark results.
//
//nolint:gocognit,maintidx,revive // ranking logic necessarily complex to handle all benchmark types
func ComputeRankings(results Results) ([]Ranking, *MedalTable) {
	scores := make(map[string]float64)
	medals := make(map[string][3]int) // [gold, silver, bronze]

	categoryMedals := make(map[string]map[string][3]int)
	categoryBenchmarks := make(map[string][]BenchmarkMedal)

	assignPoints := func(category, name string, names []string) {
		for i, n := range names {
			if i < len(placementPoints) {
				scores[n] += placementPoints[i]
			}
			if i < 3 {
				m := medals[n]
				m[i]++
				medals[n] = m

				if categoryMedals[category] == nil {
					categoryMedals[category] = make(map[string][3]int)
				}
				cm := categoryMedals[category][n]
				cm[i]++
				categoryMedals[category][n] = cm
			}
		}
		bm := BenchmarkMedal{Name: name}
		if len(names) > 0 {
			bm.Gold = names[0]
		}
		if len(names) > 1 {
			bm.Silver = names[1]
		}
		if len(names) > 2 {
			bm.Bronze = names[2]
		}
		categoryBenchmarks[category] = append(categoryBenchmarks[category], bm)
	}

	// Hit rate benchmarks - rank by average hit rate (higher is better)
	if results.HitRate != nil {
		hitRateBenchmarks := []struct {
			name string
			data []benchmark.HitRateResult
		}{
			{"CDN", results.HitRate.CDN},
			{"Meta", results.HitRate.Meta},
			{"Zipf", results.HitRate.Zipf},
			{"Twitter", results.HitRate.Twitter},
			{"Wikipedia", results.HitRate.Wikipedia},
			{"Thesios Block", results.HitRate.ThesiosBlock},
			{"Thesios File", results.HitRate.ThesiosFile},
			{"IBM Docker", results.HitRate.IBMDocker},
			{"Tencent Photo", results.HitRate.TencentPhoto},
		}
		for _, b := range hitRateBenchmarks {
			if len(b.data) == 0 {
				continue
			}
			sorted := make([]benchmark.HitRateResult, len(b.data))
			copy(sorted, b.data)
			sort.Slice(sorted, func(i, j int) bool {
				return AvgHitRate(sorted[i], results.HitRate.Sizes) > AvgHitRate(sorted[j], results.HitRate.Sizes)
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
			}
			assignPoints("Hit Rate", b.name, names)
		}
	}

	// Latency benchmarks - rank by average latency (lower is better)
	if results.Latency != nil {
		if len(results.Latency.Results) > 0 {
			sorted := make([]benchmark.LatencyResult, len(results.Latency.Results))
			copy(sorted, results.Latency.Results)
			sort.Slice(sorted, func(i, j int) bool {
				return (sorted[i].GetNsOp + sorted[i].SetNsOp) < (sorted[j].GetNsOp + sorted[j].SetNsOp)
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
			}
			assignPoints("Latency", "String Keys", names)
		}
		if len(results.Latency.IntResults) > 0 {
			sorted := make([]benchmark.LatencyResult, len(results.Latency.IntResults))
			copy(sorted, results.Latency.IntResults)
			sort.Slice(sorted, func(i, j int) bool {
				return (sorted[i].GetNsOp + sorted[i].SetNsOp) < (sorted[j].GetNsOp + sorted[j].SetNsOp)
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
			}
			assignPoints("Latency", "Int Keys", names)
		}
		if len(results.Latency.GetOrSetResults) > 0 {
			sorted := make([]benchmark.GetOrSetLatencyResult, len(results.Latency.GetOrSetResults))
			copy(sorted, results.Latency.GetOrSetResults)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].NsOp < sorted[j].NsOp
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
			}
			assignPoints("Latency", "GetOrSet", names)
		}
	}

	// Throughput benchmarks - rank by average QPS (higher is better)
	if results.Throughput != nil {
		throughputBenchmarks := []struct {
			name string
			data []benchmark.ThroughputResult
		}{
			{"String Get", results.Throughput.StringGetResults},
			{"String Set", results.Throughput.StringSetResults},
			{"Int Get", results.Throughput.IntGetResults},
			{"Int Set", results.Throughput.IntSetResults},
			{"GetOrSet", results.Throughput.GetOrSetResults},
		}
		for _, b := range throughputBenchmarks {
			if len(b.data) == 0 {
				continue
			}
			sorted := make([]benchmark.ThroughputResult, len(b.data))
			copy(sorted, b.data)
			sort.Slice(sorted, func(i, j int) bool {
				return avgQPS(sorted[i]) > avgQPS(sorted[j])
			})
			names := make([]string, len(sorted))
			for i, r := range sorted {
				names[i] = r.Name
			}
			assignPoints("Throughput", b.name, names)
		}
	}

	// Memory benchmark - rank by memory usage (lower is better)
	if results.Memory != nil && len(results.Memory.Results) > 0 {
		names := make([]string, len(results.Memory.Results))
		for i, r := range results.Memory.Results {
			names[i] = r.Name
		}
		assignPoints("Memory", "Overhead", names)
	}

	if len(scores) == 0 {
		return nil, nil
	}

	// Sort caches by score, then by medals as tiebreaker
	type cacheRank struct {
		name   string
		score  float64
		gold   int
		silver int
		bronze int
	}
	var ranks []cacheRank
	for name, score := range scores {
		m := medals[name]
		ranks = append(ranks, cacheRank{name, score, m[0], m[1], m[2]})
	}
	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].score != ranks[j].score {
			return ranks[i].score > ranks[j].score
		}
		if ranks[i].gold != ranks[j].gold {
			return ranks[i].gold > ranks[j].gold
		}
		if ranks[i].silver != ranks[j].silver {
			return ranks[i].silver > ranks[j].silver
		}
		return ranks[i].bronze > ranks[j].bronze
	})

	var result []Ranking
	for i, r := range ranks {
		result = append(result, Ranking{
			Rank:   i + 1,
			Name:   r.name,
			Score:  r.score,
			Gold:   r.gold,
			Silver: r.silver,
			Bronze: r.bronze,
		})
	}

	// Build category medal table
	catOrder := []string{"Hit Rate", "Latency", "Throughput", "Memory"}
	var categories []CategoryMedals
	for _, cat := range catOrder {
		bm := categoryBenchmarks[cat]
		if len(bm) == 0 {
			continue
		}

		cm := categoryMedals[cat]
		catRanks := make([]cacheRank, 0, len(cm))
		for name, m := range cm {
			catRanks = append(catRanks, cacheRank{
				name:   name,
				gold:   m[0],
				silver: m[1],
				bronze: m[2],
			})
		}
		sort.Slice(catRanks, func(i, j int) bool {
			if catRanks[i].gold != catRanks[j].gold {
				return catRanks[i].gold > catRanks[j].gold
			}
			if catRanks[i].silver != catRanks[j].silver {
				return catRanks[i].silver > catRanks[j].silver
			}
			return catRanks[i].bronze > catRanks[j].bronze
		})

		out := make([]Ranking, len(catRanks))
		for i, r := range catRanks {
			out[i] = Ranking{
				Rank:   i + 1,
				Name:   r.name,
				Gold:   r.gold,
				Silver: r.silver,
				Bronze: r.bronze,
			}
		}

		categories = append(categories, CategoryMedals{
			Name:       cat,
			Benchmarks: bm,
			Rankings:   out,
		})
	}

	return result, &MedalTable{Categories: categories}
}

// AvgHitRate computes the average hit rate across all cache sizes.
func AvgHitRate(r benchmark.HitRateResult, sizes []int) float64 {
	var sum float64
	for _, size := range sizes {
		sum += r.Rates[size]
	}
	return sum / float64(len(sizes))
}

func avgQPS(r benchmark.ThroughputResult) float64 {
	var sum float64
	for _, qps := range r.QPS {
		sum += qps
	}
	return sum / float64(len(r.QPS))
}
