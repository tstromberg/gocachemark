package output

import (
	"testing"

	"github.com/tstromberg/gocachemark/internal/benchmark"
)

func TestRound3(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{85.2344, 85.234},
		{85.2345, 85.235}, // rounds up
		{85.2346, 85.235},
		{0.0001, 0.0},
		{0.0005, 0.001},
		{100.0, 100.0},
		{99.9999, 100.0},
	}

	for _, tc := range tests {
		got := Round3(tc.input)
		if got != tc.expected {
			t.Errorf("Round3(%v) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestTieDetection_TwoWayGold(t *testing.T) {
	// Two caches tie for gold - both should get gold, no silver, third gets bronze
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 85.2341}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.2342}}, // ties with a (same to 3 decimals)
				{Name: "cache-c", Rates: map[int]float64{1024: 80.0}},
			},
		},
	}

	rankings, medalTable := ComputeRankings(results)

	// Check medal table
	if len(medalTable.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(medalTable.Categories))
	}

	cat := medalTable.Categories[0]
	if len(cat.Benchmarks) != 1 {
		t.Fatalf("expected 1 benchmark, got %d", len(cat.Benchmarks))
	}

	bm := cat.Benchmarks[0]

	// Both should have gold
	if len(bm.Gold) != 2 {
		t.Errorf("expected 2 gold winners, got %d: %v", len(bm.Gold), bm.Gold)
	}

	// No silver (skipped due to tie)
	if len(bm.Silver) != 0 {
		t.Errorf("expected 0 silver winners (skipped), got %d: %v", len(bm.Silver), bm.Silver)
	}

	// Third place gets bronze
	if len(bm.Bronze) != 1 || bm.Bronze[0] != "cache-c" {
		t.Errorf("expected bronze=[cache-c], got %v", bm.Bronze)
	}

	// Check rankings - both gold winners should have same points
	pointsA := findScore(rankings, "cache-a")
	pointsB := findScore(rankings, "cache-b")
	if pointsA != pointsB {
		t.Errorf("tied caches should have equal points: cache-a=%v, cache-b=%v", pointsA, pointsB)
	}
	if pointsA != 10 { // gold = 10 points
		t.Errorf("gold winners should get 10 points, got %v", pointsA)
	}
}

func TestTieDetection_ThreeWayGold(t *testing.T) {
	// Three caches tie for gold - all get gold, no silver or bronze
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 85.2340}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.2341}},
				{Name: "cache-c", Rates: map[int]float64{1024: 85.2342}},
			},
		},
	}

	_, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	if len(bm.Gold) != 3 {
		t.Errorf("expected 3 gold winners, got %d: %v", len(bm.Gold), bm.Gold)
	}
	if len(bm.Silver) != 0 {
		t.Errorf("expected 0 silver winners, got %d: %v", len(bm.Silver), bm.Silver)
	}
	if len(bm.Bronze) != 0 {
		t.Errorf("expected 0 bronze winners, got %d: %v", len(bm.Bronze), bm.Bronze)
	}
}

func TestTieDetection_TwoWaySilver(t *testing.T) {
	// Clear gold, two tie for silver, no bronze
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 90.0}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.2341}},
				{Name: "cache-c", Rates: map[int]float64{1024: 85.2342}}, // ties with b
			},
		},
	}

	_, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	if len(bm.Gold) != 1 || bm.Gold[0] != "cache-a" {
		t.Errorf("expected gold=[cache-a], got %v", bm.Gold)
	}
	if len(bm.Silver) != 2 {
		t.Errorf("expected 2 silver winners, got %d: %v", len(bm.Silver), bm.Silver)
	}
	if len(bm.Bronze) != 0 {
		t.Errorf("expected 0 bronze winners (skipped), got %d: %v", len(bm.Bronze), bm.Bronze)
	}
}

func TestTieDetection_TwoWayBronze(t *testing.T) {
	// Clear gold and silver, two tie for bronze
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 90.0}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.0}},
				{Name: "cache-c", Rates: map[int]float64{1024: 80.0001}},
				{Name: "cache-d", Rates: map[int]float64{1024: 80.0002}}, // ties with c
			},
		},
	}

	_, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	if len(bm.Gold) != 1 || bm.Gold[0] != "cache-a" {
		t.Errorf("expected gold=[cache-a], got %v", bm.Gold)
	}
	if len(bm.Silver) != 1 || bm.Silver[0] != "cache-b" {
		t.Errorf("expected silver=[cache-b], got %v", bm.Silver)
	}
	if len(bm.Bronze) != 2 {
		t.Errorf("expected 2 bronze winners, got %d: %v", len(bm.Bronze), bm.Bronze)
	}
}

func TestTieDetection_NoTies(t *testing.T) {
	// All distinct scores - normal behavior
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 90.0}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.0}},
				{Name: "cache-c", Rates: map[int]float64{1024: 80.0}},
			},
		},
	}

	rankings, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	if len(bm.Gold) != 1 || bm.Gold[0] != "cache-a" {
		t.Errorf("expected gold=[cache-a], got %v", bm.Gold)
	}
	if len(bm.Silver) != 1 || bm.Silver[0] != "cache-b" {
		t.Errorf("expected silver=[cache-b], got %v", bm.Silver)
	}
	if len(bm.Bronze) != 1 || bm.Bronze[0] != "cache-c" {
		t.Errorf("expected bronze=[cache-c], got %v", bm.Bronze)
	}

	// Verify points: gold=10, silver=7, bronze=5
	if s := findScore(rankings, "cache-a"); s != 10 {
		t.Errorf("gold should get 10 points, got %v", s)
	}
	if s := findScore(rankings, "cache-b"); s != 7 {
		t.Errorf("silver should get 7 points, got %v", s)
	}
	if s := findScore(rankings, "cache-c"); s != 5 {
		t.Errorf("bronze should get 5 points, got %v", s)
	}
}

func TestTieDetection_AlmostTied(t *testing.T) {
	// Scores differ at 3rd decimal - should NOT tie
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 85.235}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.234}}, // differs at 3rd decimal
				{Name: "cache-c", Rates: map[int]float64{1024: 80.0}},
			},
		},
	}

	_, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	// Should have distinct medals
	if len(bm.Gold) != 1 {
		t.Errorf("expected 1 gold winner, got %d: %v", len(bm.Gold), bm.Gold)
	}
	if len(bm.Silver) != 1 {
		t.Errorf("expected 1 silver winner, got %d: %v", len(bm.Silver), bm.Silver)
	}
	if len(bm.Bronze) != 1 {
		t.Errorf("expected 1 bronze winner, got %d: %v", len(bm.Bronze), bm.Bronze)
	}
}

func findScore(rankings []Ranking, name string) float64 {
	for _, r := range rankings {
		if r.Name == name {
			return r.Score
		}
	}
	return -1
}

func findRanking(rankings []Ranking, name string) *Ranking {
	for _, r := range rankings {
		if r.Name == name {
			return &r
		}
	}
	return nil
}

func TestTieDetection_Latency(t *testing.T) {
	// Latency is lower-is-better, verify ties work with reversed sorting
	results := Results{
		Latency: &LatencyData{
			Results: []benchmark.LatencyResult{
				{Name: "cache-a", GetNsOp: 10.0, SetNsOp: 10.0},    // avg 10
				{Name: "cache-b", GetNsOp: 10.001, SetNsOp: 9.999}, // avg 10 (ties with a)
				{Name: "cache-c", GetNsOp: 20.0, SetNsOp: 20.0},    // avg 20
			},
		},
	}

	_, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	if len(bm.Gold) != 2 {
		t.Errorf("expected 2 gold winners for latency tie, got %d: %v", len(bm.Gold), bm.Gold)
	}
	if len(bm.Silver) != 0 {
		t.Errorf("expected 0 silver (skipped), got %d: %v", len(bm.Silver), bm.Silver)
	}
	if len(bm.Bronze) != 1 || bm.Bronze[0] != "cache-c" {
		t.Errorf("expected bronze=[cache-c], got %v", bm.Bronze)
	}
}

func TestTieDetection_Throughput(t *testing.T) {
	// Throughput is higher-is-better
	results := Results{
		Throughput: &ThroughputData{
			Threads: []int{1},
			StringGetResults: []benchmark.ThroughputResult{
				{Name: "cache-a", QPS: map[int]float64{1: 1000000.0}},
				{Name: "cache-b", QPS: map[int]float64{1: 1000000.0001}}, // ties with a
				{Name: "cache-c", QPS: map[int]float64{1: 500000.0}},
			},
		},
	}

	_, medalTable := ComputeRankings(results)
	bm := medalTable.Categories[0].Benchmarks[0]

	if len(bm.Gold) != 2 {
		t.Errorf("expected 2 gold winners for throughput tie, got %d: %v", len(bm.Gold), bm.Gold)
	}
}

func TestMedalAccumulation(t *testing.T) {
	// Cache wins gold in one benchmark, silver in another
	// Verify medal counts are accumulated correctly
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 90.0}}, // gold
				{Name: "cache-b", Rates: map[int]float64{1024: 80.0}}, // silver
			},
			CDN: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 70.0}}, // silver
				{Name: "cache-b", Rates: map[int]float64{1024: 80.0}}, // gold
			},
		},
	}

	rankings, _ := ComputeRankings(results)

	rA := findRanking(rankings, "cache-a")
	rB := findRanking(rankings, "cache-b")

	if rA == nil || rB == nil {
		t.Fatal("rankings not found")
	}

	// Each should have 1 gold, 1 silver
	if rA.Gold != 1 || rA.Silver != 1 {
		t.Errorf("cache-a: expected 1 gold, 1 silver; got %d gold, %d silver", rA.Gold, rA.Silver)
	}
	if rB.Gold != 1 || rB.Silver != 1 {
		t.Errorf("cache-b: expected 1 gold, 1 silver; got %d gold, %d silver", rB.Gold, rB.Silver)
	}

	// Both should have same score: 10 (gold) + 7 (silver) = 17
	if rA.Score != 17 || rB.Score != 17 {
		t.Errorf("expected both scores=17, got cache-a=%v, cache-b=%v", rA.Score, rB.Score)
	}
}

func TestTieDetection_EmptyResults(t *testing.T) {
	results := Results{}

	rankings, medalTable := ComputeRankings(results)

	if rankings != nil {
		t.Errorf("expected nil rankings for empty results, got %v", rankings)
	}
	if medalTable != nil {
		t.Errorf("expected nil medalTable for empty results, got %v", medalTable)
	}
}

func TestTieDetection_SingleCache(t *testing.T) {
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "only-cache", Rates: map[int]float64{1024: 85.0}},
			},
		},
	}

	rankings, medalTable := ComputeRankings(results)

	if len(rankings) != 1 {
		t.Fatalf("expected 1 ranking, got %d", len(rankings))
	}
	if rankings[0].Gold != 1 {
		t.Errorf("single cache should get gold, got %d golds", rankings[0].Gold)
	}

	bm := medalTable.Categories[0].Benchmarks[0]
	if len(bm.Gold) != 1 || bm.Gold[0] != "only-cache" {
		t.Errorf("expected gold=[only-cache], got %v", bm.Gold)
	}
}

func TestTieDetection_AllTied(t *testing.T) {
	// All caches have identical scores
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 85.0}},
				{Name: "cache-b", Rates: map[int]float64{1024: 85.0}},
				{Name: "cache-c", Rates: map[int]float64{1024: 85.0}},
				{Name: "cache-d", Rates: map[int]float64{1024: 85.0}},
			},
		},
	}

	rankings, medalTable := ComputeRankings(results)

	// All should get gold
	bm := medalTable.Categories[0].Benchmarks[0]
	if len(bm.Gold) != 4 {
		t.Errorf("expected 4 gold winners, got %d: %v", len(bm.Gold), bm.Gold)
	}
	if len(bm.Silver) != 0 || len(bm.Bronze) != 0 {
		t.Errorf("expected no silver/bronze, got silver=%v, bronze=%v", bm.Silver, bm.Bronze)
	}

	// All should have same score (10 points for gold)
	for _, r := range rankings {
		if r.Score != 10 {
			t.Errorf("%s: expected score=10, got %v", r.Name, r.Score)
		}
		if r.Gold != 1 {
			t.Errorf("%s: expected 1 gold, got %d", r.Name, r.Gold)
		}
	}
}

func TestTieDetection_FourthPlaceAfterThreeWayTie(t *testing.T) {
	// Three-way tie for gold, fourth place should get no medal but still get points
	results := Results{
		HitRate: &HitRateData{
			Sizes: []int{1024},
			Zipf: []benchmark.HitRateResult{
				{Name: "cache-a", Rates: map[int]float64{1024: 90.0}},
				{Name: "cache-b", Rates: map[int]float64{1024: 90.0}},
				{Name: "cache-c", Rates: map[int]float64{1024: 90.0}},
				{Name: "cache-d", Rates: map[int]float64{1024: 80.0}}, // 4th place
			},
		},
	}

	rankings, medalTable := ComputeRankings(results)

	bm := medalTable.Categories[0].Benchmarks[0]

	// Three golds, no silver/bronze
	if len(bm.Gold) != 3 {
		t.Errorf("expected 3 gold, got %d", len(bm.Gold))
	}
	if len(bm.Silver) != 0 || len(bm.Bronze) != 0 {
		t.Errorf("expected no silver/bronze after 3-way gold tie")
	}

	// Fourth place gets 4th place points (4 points) but no medal
	rD := findRanking(rankings, "cache-d")
	if rD == nil {
		t.Fatal("cache-d not found in rankings")
	}
	if rD.Gold != 0 || rD.Silver != 0 || rD.Bronze != 0 {
		t.Errorf("cache-d should have no medals, got g=%d s=%d b=%d", rD.Gold, rD.Silver, rD.Bronze)
	}
	if rD.Score != 4 { // 4th place = 4 points
		t.Errorf("cache-d should have 4 points (4th place), got %v", rD.Score)
	}
}
