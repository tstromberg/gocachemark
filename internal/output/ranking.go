package output

import (
	"math"
	"sort"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

// Points awarded by placement: 1st=10, 2nd=7, 3rd=5, 4th=4, 5th=3, 6th=2, 7th=1.
var placementPoints = []float64{10, 7, 5, 4, 3, 2, 1}

// rankedEntry holds a name and score for tie detection.
type rankedEntry struct {
	name  string
	score float64
}

// Round3 rounds to 3 decimal places for tie detection.
func Round3(f float64) float64 {
	return math.Round(f*1000) / 1000
}

// WinnerEntry represents a ranked entry for winner display.
type WinnerEntry struct {
	Name  string
	Score float64
}

// FormatWinners returns winner names and the first runner-up for comparison.
// If multiple entries tie for first, all are returned as winners.
// Returns (winners, runnerUp) where runnerUp is nil if everyone ties or only one entry.
func FormatWinners(entries []WinnerEntry) (winners []string, runnerUp *WinnerEntry) {
	if len(entries) == 0 {
		return nil, nil
	}

	// Find all entries tied for first
	bestScore := Round3(entries[0].Score)
	for _, e := range entries {
		if Round3(e.Score) != bestScore {
			runnerUp = &WinnerEntry{Name: e.Name, Score: e.Score}
			break
		}
		winners = append(winners, e.Name)
	}

	return winners, runnerUp
}

// ComputeRankings calculates overall rankings from benchmark results.
//
//nolint:gocognit,maintidx,revive // ranking logic necessarily complex to handle all benchmark types
func ComputeRankings(results Results) ([]Ranking, *MedalTable) {
	scores := make(map[string]float64)
	medals := make(map[string][3]int) // [gold, silver, bronze]

	categoryMedals := make(map[string]map[string][3]int)
	categoryBenchmarks := make(map[string][]BenchmarkMedal)

	// assignPoints handles tie detection: entries with scores equal to 3 decimal
	// places share the same medal position. Entries must be pre-sorted by score.
	assignPoints := func(category, benchName string, entries []rankedEntry) {
		bm := BenchmarkMedal{Name: benchName}
		pos := 0 // current medal position (0=gold, 1=silver, 2=bronze)
		i := 0

		for i < len(entries) {
			// Find all entries tied at this position
			var tied []string
			baseScore := Round3(entries[i].score)
			for i < len(entries) && Round3(entries[i].score) == baseScore {
				tied = append(tied, entries[i].name)
				i++
			}

			// Assign points and medals to all tied entries
			for _, n := range tied {
				if pos < len(placementPoints) {
					scores[n] += placementPoints[pos]
				}
				if pos < 3 {
					m := medals[n]
					m[pos]++
					medals[n] = m

					if categoryMedals[category] == nil {
						categoryMedals[category] = make(map[string][3]int)
					}
					cm := categoryMedals[category][n]
					cm[pos]++
					categoryMedals[category][n] = cm
				}
			}

			// Store tied winners in medal struct
			if pos < 3 {
				switch pos {
				case 0:
					bm.Gold = tied
				case 1:
					bm.Silver = tied
				case 2:
					bm.Bronze = tied
				}
			}

			// Skip positions based on number of ties
			pos += len(tied)
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
			entries := make([]rankedEntry, len(sorted))
			for i, r := range sorted {
				entries[i] = rankedEntry{r.Name, AvgHitRate(r, results.HitRate.Sizes)}
			}
			assignPoints("Hit Rate", b.name, entries)
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
			entries := make([]rankedEntry, len(sorted))
			for i, r := range sorted {
				entries[i] = rankedEntry{r.Name, (r.GetNsOp + r.SetNsOp) / 2}
			}
			assignPoints("Latency", "String Keys", entries)
		}
		if len(results.Latency.IntResults) > 0 {
			sorted := make([]benchmark.LatencyResult, len(results.Latency.IntResults))
			copy(sorted, results.Latency.IntResults)
			sort.Slice(sorted, func(i, j int) bool {
				return (sorted[i].GetNsOp + sorted[i].SetNsOp) < (sorted[j].GetNsOp + sorted[j].SetNsOp)
			})
			entries := make([]rankedEntry, len(sorted))
			for i, r := range sorted {
				entries[i] = rankedEntry{r.Name, (r.GetNsOp + r.SetNsOp) / 2}
			}
			assignPoints("Latency", "Int Keys", entries)
		}
		if len(results.Latency.GetOrSetResults) > 0 {
			sorted := make([]benchmark.GetOrSetLatencyResult, len(results.Latency.GetOrSetResults))
			copy(sorted, results.Latency.GetOrSetResults)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].NsOp < sorted[j].NsOp
			})
			entries := make([]rankedEntry, len(sorted))
			for i, r := range sorted {
				entries[i] = rankedEntry{r.Name, r.NsOp}
			}
			assignPoints("Latency", "GetOrSet", entries)
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
			entries := make([]rankedEntry, len(sorted))
			for i, r := range sorted {
				entries[i] = rankedEntry{r.Name, avgQPS(r)}
			}
			assignPoints("Throughput", b.name, entries)
		}
	}

	// Memory benchmark - rank by memory usage (lower is better)
	if results.Memory != nil && len(results.Memory.Results) > 0 {
		entries := make([]rankedEntry, len(results.Memory.Results))
		for i, r := range results.Memory.Results {
			entries[i] = rankedEntry{r.Name, float64(r.Bytes)}
		}
		assignPoints("Memory", "Overhead", entries)
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
