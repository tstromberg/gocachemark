// Package workload generates cache workload patterns.
package workload

import (
	"math"
	"math/rand/v2"
)

// GenerateZipfInt generates a Zipfian distribution of keys as integers.
// n is the number of keys to generate, keySpace is the range of keys,
// theta controls the skew (higher = more skewed), seed is for reproducibility.
func GenerateZipfInt(n, keySpace int, theta float64, seed uint64) []int {
	rng := rand.New(rand.NewPCG(seed, seed+1))
	keys := make([]int, n)

	spread := keySpace + 1
	zeta2 := computeZeta(2, theta)
	zetaN := computeZeta(uint64(spread), theta)
	alpha := 1.0 / (1.0 - theta)
	eta := (1 - math.Pow(2.0/float64(spread), 1.0-theta)) / (1.0 - zeta2/zetaN)
	halfPowTheta := 1.0 + math.Pow(0.5, theta)

	for i := range n {
		u := rng.Float64()
		uz := u * zetaN
		var result int
		switch {
		case uz < 1.0:
			result = 0
		case uz < halfPowTheta:
			result = 1
		default:
			result = int(float64(spread) * math.Pow(eta*u-eta+1.0, alpha))
		}
		if result >= keySpace {
			result = keySpace - 1
		}
		keys[i] = result
	}
	return keys
}

func computeZeta(n uint64, theta float64) float64 {
	sum := 0.0
	for i := uint64(1); i <= n; i++ {
		sum += 1.0 / math.Pow(float64(i), theta)
	}
	return sum
}
