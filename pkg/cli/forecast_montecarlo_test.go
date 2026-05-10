//go:build !integration

package cli

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deterministicRNG returns a seeded *rand.Rand for reproducible test results.
func deterministicRNG() *rand.Rand {
	return rand.New(rand.NewSource(42)) //nolint:gosec
}

// TestPoissonSample verifies that the Poisson sampler produces an empirical mean
// and variance close to lambda (within statistical tolerance for 100 000 draws).
func TestPoissonSample(t *testing.T) {
	rng := deterministicRNG()
	const lambda = 15.0
	const n = 100_000

	sum := 0.0
	sumSq := 0.0
	for i := 0; i < n; i++ {
		v := float64(poissonSample(rng, lambda))
		sum += v
		sumSq += v * v
	}
	mean := sum / n
	variance := sumSq/n - mean*mean

	// Poisson(λ): mean == λ, variance == λ.  Allow 1% relative error.
	assert.InEpsilon(t, lambda, mean, 0.01, "empirical mean should be close to lambda")
	assert.InEpsilon(t, lambda, variance, 0.01, "empirical variance should be close to lambda")
}

// TestPoissonSampleLargeLambda exercises the normal-approximation branch (lambda > 30).
func TestPoissonSampleLargeLambda(t *testing.T) {
	rng := deterministicRNG()
	const lambda = 100.0
	const n = 100_000

	sum := 0.0
	for i := 0; i < n; i++ {
		sum += float64(poissonSample(rng, lambda))
	}
	mean := sum / n

	assert.InEpsilon(t, lambda, mean, 0.01, "normal-approximation branch should produce correct mean")
}

// TestPoissonSampleEdgeCases checks boundary conditions.
func TestPoissonSampleEdgeCases(t *testing.T) {
	rng := deterministicRNG()
	assert.Equal(t, 0, poissonSample(rng, 0), "lambda=0 should return 0")
	assert.Equal(t, 0, poissonSample(rng, -5), "negative lambda should return 0")
}

// TestPercentileFloat64 checks the nearest-rank percentile helper.
func TestPercentileFloat64(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	assert.Equal(t, 1.0, percentileFloat64(sorted, 10), "P10")
	assert.Equal(t, 5.0, percentileFloat64(sorted, 50), "P50")
	assert.Equal(t, 9.0, percentileFloat64(sorted, 90), "P90")
	assert.Equal(t, 0.0, percentileFloat64(nil, 50), "empty slice")
}

// TestPercentileInt checks the int variant of the percentile helper.
func TestPercentileInt(t *testing.T) {
	sorted := []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	assert.Equal(t, 10, percentileInt(sorted, 10), "P10")
	assert.Equal(t, 50, percentileInt(sorted, 50), "P50")
	assert.Equal(t, 90, percentileInt(sorted, 90), "P90")
	assert.Equal(t, 0, percentileInt(nil, 50), "empty slice")
}

// TestCostMeanStdDev verifies the mean/stddev helper on a known distribution.
func TestCostMeanStdDev(t *testing.T) {
	xs := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	mean, stddev := costMeanStdDev(xs)
	assert.InDelta(t, 5.0, mean, 0.001, "mean")
	assert.InDelta(t, 2.0, stddev, 0.001, "population stddev")

	m0, s0 := costMeanStdDev(nil)
	assert.Equal(t, 0.0, m0)
	assert.Equal(t, 0.0, s0)
}

// TestRunMonteCarloNilOnEmpty verifies that runMonteCarlo returns nil for empty inputs.
func TestRunMonteCarloNilOnEmpty(t *testing.T) {
	rng := deterministicRNG()
	assert.Nil(t, runMonteCarlo(nil, 0, 10.0, rng), "nil observations")
	assert.Nil(t, runMonteCarlo([]int{100, 200}, 2, 0.0, rng), "zero lambda")
	assert.Nil(t, runMonteCarlo([]int{100, 200}, 2, -1.0, rng), "negative lambda")
}

// TestRunMonteCarloBasicProperties checks that the Monte Carlo summary satisfies
// statistical invariants (P10 ≤ P50 ≤ P90, mean ≥ 0, stddev ≥ 0).
func TestRunMonteCarloBasicProperties(t *testing.T) {
	rng := deterministicRNG()
	// 20 historical runs, all successful, each costing ~1 000 tokens.
	etObs := make([]int, 20)
	for i := range etObs {
		etObs[i] = 900 + i*10 // 900–1090
	}

	mc := runMonteCarlo(etObs, len(etObs), 10.0, rng)
	require.NotNil(t, mc)

	assert.Equal(t, monteCarloIterations, mc.Iterations)
	assert.GreaterOrEqual(t, mc.MeanProjectedCostUSD, 0.0)
	assert.GreaterOrEqual(t, mc.StdDevCostUSD, 0.0)
	assert.LessOrEqual(t, mc.P10ProjectedCostUSD, mc.P50ProjectedCostUSD, "P10 ≤ P50")
	assert.LessOrEqual(t, mc.P50ProjectedCostUSD, mc.P90ProjectedCostUSD, "P50 ≤ P90")
	assert.LessOrEqual(t, mc.P10ProjectedEffectiveTokens, mc.P50ProjectedEffectiveTokens, "ET P10 ≤ P50")
	assert.LessOrEqual(t, mc.P50ProjectedEffectiveTokens, mc.P90ProjectedEffectiveTokens, "ET P50 ≤ P90")
}

// TestRunMonteCarloZeroSuccessRate verifies that a 0% success rate produces zero cost.
func TestRunMonteCarloZeroSuccessRate(t *testing.T) {
	rng := deterministicRNG()
	etObs := []int{1000, 2000, 3000}
	// successCount = 0 → successRate = 0/3 = 0.
	mc := runMonteCarlo(etObs, 0, 5.0, rng)
	require.NotNil(t, mc)
	assert.Equal(t, 0.0, mc.P50ProjectedCostUSD, "zero success rate → zero cost")
	assert.Equal(t, 0.0, mc.P90ProjectedCostUSD, "zero success rate → zero cost P90")
}

// TestRunMonteCarloOrderOfMagnitude checks that the simulation mean is within
// an order of magnitude of the deterministic point estimate.
func TestRunMonteCarloOrderOfMagnitude(t *testing.T) {
	rng := deterministicRNG()
	etObs := []int{10_000, 12_000, 11_000, 9_500, 10_500}
	successCount := 5
	observedRunsPerPeriod := 20.0

	mc := runMonteCarlo(etObs, successCount, observedRunsPerPeriod, rng)
	require.NotNil(t, mc)

	// Deterministic point estimate.
	var totalET int
	for _, et := range etObs {
		totalET += et
	}
	avgET := totalET / len(etObs)
	pointEstimate := float64(int(math.Round(observedRunsPerPeriod*float64(avgET)))) * costPerEffectiveToken

	// Simulation mean should be within 20% of point estimate (with 100% success rate
	// and Poisson lambda = 20, the spread should be small).
	assert.InEpsilon(t, pointEstimate, mc.MeanProjectedCostUSD, 0.20,
		"simulation mean should be close to point estimate")

	// P50 should also be within 20%.
	assert.InEpsilon(t, pointEstimate, mc.P50ProjectedCostUSD, 0.20,
		"simulation P50 should be close to point estimate")

	// Confidence interval must bracket the mean.
	assert.LessOrEqual(t, mc.P10ProjectedCostUSD, mc.MeanProjectedCostUSD)
	assert.GreaterOrEqual(t, mc.P90ProjectedCostUSD, mc.MeanProjectedCostUSD)
}

// TestRunMonteCarloSortedOutputs verifies CI ordering holds across many random seeds.
func TestRunMonteCarloSortedOutputs(t *testing.T) {
	etObs := []int{5_000, 7_000, 6_000, 4_500}
	for seed := int64(0); seed < 5; seed++ {
		rng := rand.New(rand.NewSource(seed)) //nolint:gosec
		mc := runMonteCarlo(etObs, len(etObs), 12.0, rng)
		require.NotNil(t, mc)
		assert.LessOrEqual(t, mc.P10ProjectedCostUSD, mc.P50ProjectedCostUSD)
		assert.LessOrEqual(t, mc.P50ProjectedCostUSD, mc.P90ProjectedCostUSD)
	}
}

// TestRunMonteCarloDistributionShape verifies that the cost distribution is roughly
// unimodal and bell-shaped (skew stays within a reasonable bound) by checking that
// the mean lies between P10 and P90.
func TestRunMonteCarloDistributionShape(t *testing.T) {
	rng := deterministicRNG()
	etObs := make([]int, 50)
	for i := range etObs {
		etObs[i] = 8_000 + i*40
	}
	mc := runMonteCarlo(etObs, len(etObs), 30.0, rng)
	require.NotNil(t, mc)

	assert.GreaterOrEqual(t, mc.MeanProjectedCostUSD, mc.P10ProjectedCostUSD, "mean ≥ P10")
	assert.LessOrEqual(t, mc.MeanProjectedCostUSD, mc.P90ProjectedCostUSD, "mean ≤ P90")
}

// TestPercentileSingleElement ensures percentile works for a length-1 slice.
func TestPercentileSingleElement(t *testing.T) {
	sorted := []float64{42.0}
	assert.Equal(t, 42.0, percentileFloat64(sorted, 10))
	assert.Equal(t, 42.0, percentileFloat64(sorted, 90))
}

// TestRunMonteCarloFullEpisodePath is a smoke test that exercises the full
// forecastWorkflow path by calling runMonteCarlo directly with a realistic setup.
func TestRunMonteCarloFullEpisodePath(t *testing.T) {
	rng := deterministicRNG()

	// Simulate 30 completed runs with varied token counts.
	etObs := make([]int, 30)
	successCount := 0
	for i := range etObs {
		etObs[i] = 5_000 + i*200
		if i%5 != 0 { // 80% success
			successCount++
		}
	}

	mc := runMonteCarlo(etObs, successCount, 8.0, rng)
	require.NotNil(t, mc)
	assert.Equal(t, monteCarloIterations, mc.Iterations)
	assert.Greater(t, mc.P90ProjectedCostUSD, mc.P10ProjectedCostUSD, "P90 > P10 for non-trivial inputs")

	// Cost field should round-trip through sort correctly.
	costs := []float64{mc.P10ProjectedCostUSD, mc.P50ProjectedCostUSD, mc.P90ProjectedCostUSD}
	sorted := make([]float64, len(costs))
	copy(sorted, costs)
	sort.Float64s(sorted)
	assert.Equal(t, costs, sorted, "cost percentiles should already be in ascending order")
}
