//go:build !integration

package cli

import (
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

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
	const lambda = 10.0 // within Knuth's exact branch (≤15)
	const n = 100_000

	sum := 0.0
	sumSq := 0.0
	for range n {
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

// TestPoissonSampleLargeLambda exercises the normal-approximation branch (lambda > 15).
func TestPoissonSampleLargeLambda(t *testing.T) {
	rng := deterministicRNG()
	const lambda = 100.0
	const n = 100_000

	sum := 0.0
	for range n {
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

func TestUseNormalApproximationForPoissonThreshold(t *testing.T) {
	assert.False(t, useNormalApproximationForPoisson(poissonNormalApproximationThreshold), "lambda at threshold should use Knuth exact branch")
	assert.True(t, useNormalApproximationForPoisson(poissonNormalApproximationThreshold+0.0001), "lambda above threshold should use Normal approximation")
}

// TestPercentileInt checks the int variant of the percentile helper.
func TestPercentileInt(t *testing.T) {
	sorted := []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	assert.Equal(t, 10, percentileInt(sorted, 10), "P10")
	assert.Equal(t, 50, percentileInt(sorted, 50), "P50")
	assert.Equal(t, 90, percentileInt(sorted, 90), "P90")
	assert.Equal(t, 0, percentileInt(nil, 50), "empty slice")
}

// TestMeanStdDevInt verifies the mean/stddev helper on a known distribution.
func TestMeanStdDevInt(t *testing.T) {
	// population stddev of {2,4,4,4,5,5,7,9} = 2, mean = 5.
	xs := []int{2, 4, 4, 4, 5, 5, 7, 9}
	mean, stddev := meanStdDevInt(xs)
	assert.Equal(t, 5, mean, "mean")
	assert.InDelta(t, 2.0, stddev, 0.001, "population stddev")

	m0, s0 := meanStdDevInt(nil)
	assert.Equal(t, 0, m0)
	assert.InDelta(t, 0.0, s0, 0)
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
	// 20 historical runs, all successful, each using ~1 000 tokens.
	etObs := make([]int, 20)
	for i := range etObs {
		etObs[i] = 900 + i*10 // 900–1090
	}

	mc := runMonteCarlo(etObs, len(etObs), 10.0, rng)
	require.NotNil(t, mc)

	assert.Equal(t, monteCarloIterations, mc.Iterations)
	assert.GreaterOrEqual(t, mc.MeanProjectedEffectiveTokens, 0)
	assert.GreaterOrEqual(t, mc.StdDevEffectiveTokens, 0.0)
	assert.LessOrEqual(t, mc.P10ProjectedEffectiveTokens, mc.P50ProjectedEffectiveTokens, "ET P10 ≤ P50")
	assert.LessOrEqual(t, mc.P50ProjectedEffectiveTokens, mc.P90ProjectedEffectiveTokens, "ET P50 ≤ P90")
}

// TestRunMonteCarloZeroSuccessRate verifies that a 0% success rate produces zero ET.
func TestRunMonteCarloZeroSuccessRate(t *testing.T) {
	rng := deterministicRNG()
	etObs := []int{1000, 2000, 3000}
	// successCount = 0 → successRate = 0/3 = 0.
	mc := runMonteCarlo(etObs, 0, 5.0, rng)
	require.NotNil(t, mc)
	assert.Equal(t, 0, mc.P50ProjectedEffectiveTokens, "zero success rate → zero ET")
	assert.Equal(t, 0, mc.P90ProjectedEffectiveTokens, "zero success rate → zero ET P90")
}

// TestRunMonteCarloOrderOfMagnitude checks that the simulation mean is within
// 20% of the deterministic point estimate.
func TestRunMonteCarloOrderOfMagnitude(t *testing.T) {
	rng := deterministicRNG()
	etObs := []int{10_000, 12_000, 11_000, 9_500, 10_500}
	successCount := 5
	observedRunsPerPeriod := 20.0

	mc := runMonteCarlo(etObs, successCount, observedRunsPerPeriod, rng)
	require.NotNil(t, mc)

	// Deterministic point estimate (ET).
	var totalET int
	for _, et := range etObs {
		totalET += et
	}
	avgET := totalET / len(etObs)
	pointEstimate := int(math.Round(observedRunsPerPeriod * float64(avgET)))

	// Simulation mean should be within 20% of point estimate (with 100% success rate
	// and Poisson lambda = 20, the spread should be small).
	assert.InEpsilon(t, float64(pointEstimate), float64(mc.MeanProjectedEffectiveTokens), 0.20,
		"simulation mean ET should be close to point estimate")

	// P50 should also be within 20%.
	assert.InEpsilon(t, float64(pointEstimate), float64(mc.P50ProjectedEffectiveTokens), 0.20,
		"simulation P50 ET should be close to point estimate")

	// Confidence interval must bracket the mean.
	assert.LessOrEqual(t, mc.P10ProjectedEffectiveTokens, mc.MeanProjectedEffectiveTokens)
	assert.GreaterOrEqual(t, mc.P90ProjectedEffectiveTokens, mc.MeanProjectedEffectiveTokens)
}

// TestRunMonteCarloSortedOutputs verifies CI ordering holds across many random seeds.
func TestRunMonteCarloSortedOutputs(t *testing.T) {
	etObs := []int{5_000, 7_000, 6_000, 4_500}
	for seed := range 5 {
		rng := rand.New(rand.NewSource(int64(seed))) //nolint:gosec
		mc := runMonteCarlo(etObs, len(etObs), 12.0, rng)
		require.NotNil(t, mc)
		assert.LessOrEqual(t, mc.P10ProjectedEffectiveTokens, mc.P50ProjectedEffectiveTokens)
		assert.LessOrEqual(t, mc.P50ProjectedEffectiveTokens, mc.P90ProjectedEffectiveTokens)
	}
}

// TestRunMonteCarloDistributionShape verifies that the ET distribution is roughly
// unimodal by checking that the mean lies between P10 and P90.
func TestRunMonteCarloDistributionShape(t *testing.T) {
	rng := deterministicRNG()
	etObs := make([]int, 50)
	for i := range etObs {
		etObs[i] = 8_000 + i*40
	}
	mc := runMonteCarlo(etObs, len(etObs), 30.0, rng)
	require.NotNil(t, mc)

	assert.GreaterOrEqual(t, mc.MeanProjectedEffectiveTokens, mc.P10ProjectedEffectiveTokens, "mean ≥ P10")
	assert.LessOrEqual(t, mc.MeanProjectedEffectiveTokens, mc.P90ProjectedEffectiveTokens, "mean ≤ P90")
}

// TestGammaSampleMeanVariance verifies that gammaSample produces the expected mean
// (= shape) and variance (= shape) for a Gamma(shape, scale=1) distribution.
func TestGammaSampleMeanVariance(t *testing.T) {
	rng := deterministicRNG()
	const shape = 5.5 // typical value: n+0.5 for n=5 observed runs
	const n = 200_000

	var sum, sumSq float64
	for range n {
		v := gammaSample(rng, shape)
		sum += v
		sumSq += v * v
	}
	mean := sum / n
	variance := sumSq/n - mean*mean

	// Gamma(shape, scale=1): mean = shape, variance = shape.  Allow 1% relative error.
	assert.InEpsilon(t, shape, mean, 0.01, "gamma empirical mean should equal shape")
	assert.InEpsilon(t, shape, variance, 0.01, "gamma empirical variance should equal shape")
}

// TestGammaSampleSmallShape verifies the shape < 1 reduction path for multiple
// fractional shape values (0.3, 0.5, 0.8) to ensure the recursive identity
// Gamma(shape) = Gamma(shape+1) × U^(1/shape) is exercised correctly.
func TestGammaSampleSmallShape(t *testing.T) {
	const n = 200_000
	for _, shape := range []float64{0.3, 0.5, 0.8} {
		rng := deterministicRNG()
		var sum float64
		for range n {
			sum += gammaSample(rng, shape)
		}
		mean := sum / n
		assert.InEpsilon(t, shape, mean, 0.01,
			"gamma mean should equal shape for shape=%v", shape)
	}
}

// TestGammaSampleEdgeCases checks boundary and degenerate inputs.
func TestGammaSampleEdgeCases(t *testing.T) {
	rng := deterministicRNG()
	assert.InDelta(t, 0.0, gammaSample(rng, 0), 0, "shape=0 → 0")
	assert.InDelta(t, 0.0, gammaSample(rng, -1), 0, "shape<0 → 0")
}

// TestRunMonteCarloIsReliable verifies that IsReliable reflects the minimum
// observation threshold.
func TestRunMonteCarloIsReliable(t *testing.T) {
	rng := deterministicRNG()

	// Below threshold: 3 observations < minObservationsForReliableForecast (10).
	smallObs := []int{1000, 1500, 1200}
	mcSmall := runMonteCarlo(smallObs, len(smallObs), 4.0, rng)
	require.NotNil(t, mcSmall)
	assert.False(t, mcSmall.IsReliable, "fewer than 10 observations → IsReliable=false")

	// At threshold: exactly minObservationsForReliableForecast observations.
	atThreshold := []int{1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900}
	mcAt := runMonteCarlo(atThreshold, len(atThreshold), 4.0, rng)
	require.NotNil(t, mcAt)
	assert.True(t, mcAt.IsReliable, "exactly 10 observations → IsReliable=true")

	// Well above threshold.
	largeObs := make([]int, 20)
	for i := range largeObs {
		largeObs[i] = 1000 + i*50
	}
	mcLarge := runMonteCarlo(largeObs, len(largeObs), 10.0, rng)
	require.NotNil(t, mcLarge)
	assert.True(t, mcLarge.IsReliable, "20 observations → IsReliable=true")
}

// TestRunMonteCarloGammaPoissonWiderCI verifies that the Gamma–Poisson compound model
// produces wider confidence intervals for small samples compared to a scenario where
// the rate is well-estimated (large sample).  With small n the posterior Gamma has
// higher relative variance, so the simulated ET distribution should be broader.
func TestRunMonteCarloGammaPoissonWiderCI(t *testing.T) {
	// Same observed rate (λ = 10) but different sample sizes.
	etVal := 1_000 // constant ET to isolate run-count variability
	const lambda = 10.0

	// Small sample: 3 runs observed → high relative uncertainty in λ.
	smallObs := []int{etVal, etVal, etVal}
	rngSmall := rand.New(rand.NewSource(7)) //nolint:gosec
	mcSmall := runMonteCarlo(smallObs, len(smallObs), lambda, rngSmall)
	require.NotNil(t, mcSmall)

	// Large sample: 100 runs observed → low relative uncertainty in λ.
	largeObs := make([]int, 100)
	for i := range largeObs {
		largeObs[i] = etVal
	}
	rngLarge := rand.New(rand.NewSource(7)) //nolint:gosec
	mcLarge := runMonteCarlo(largeObs, len(largeObs), lambda, rngLarge)
	require.NotNil(t, mcLarge)

	ciSmall := mcSmall.P90ProjectedEffectiveTokens - mcSmall.P10ProjectedEffectiveTokens
	ciLarge := mcLarge.P90ProjectedEffectiveTokens - mcLarge.P10ProjectedEffectiveTokens

	assert.Greater(t, ciSmall, ciLarge,
		"small-sample CI (P90-P10=%d) should be wider than large-sample CI (%d)", ciSmall, ciLarge)
}

// TestRunMonteCarloFullEpisodePath is a smoke test that exercises runMonteCarlo
// with a realistic setup and validates ET percentile ordering.
func TestRunMonteCarloFullEpisodePath(t *testing.T) {
	rng := deterministicRNG()

	// Simulate 30 completed runs with varied token counts.
	etObs := make([]int, 30)
	successCount := 0
	for i := range etObs {
		etObs[i] = 5_000 + i*200
		if i%5 != 0 { // 4 out of every 5 runs succeed → 80% success rate
			successCount++
		}
	}

	mc := runMonteCarlo(etObs, successCount, 8.0, rng)
	require.NotNil(t, mc)
	assert.Equal(t, monteCarloIterations, mc.Iterations)
	assert.Greater(t, mc.P90ProjectedEffectiveTokens, mc.P10ProjectedEffectiveTokens, "P90 > P10 for non-trivial inputs")

	// ET percentiles should already be in ascending order.
	ets := []int{mc.P10ProjectedEffectiveTokens, mc.P50ProjectedEffectiveTokens, mc.P90ProjectedEffectiveTokens}
	sorted := make([]int, len(ets))
	copy(sorted, ets)
	sort.Ints(sorted)
	assert.Equal(t, ets, sorted, "ET percentiles should already be in ascending order")
}

func TestResolveForecastWorkflowsFromRemote_RateLimitFallsBackToPartialResults(t *testing.T) {
	originalFetch := forecastFetchGitHubWorkflows
	originalSleep := forecastRateLimitSleep
	t.Cleanup(func() {
		forecastFetchGitHubWorkflows = originalFetch
		forecastRateLimitSleep = originalSleep
	})

	attempts := 0
	var backoffs []time.Duration
	forecastFetchGitHubWorkflows = func(repoOverride string, verbose bool) (map[string]*GitHubWorkflow, error) {
		attempts++
		return nil, errors.New("API rate limit exceeded")
	}
	forecastRateLimitSleep = func(delay time.Duration) {
		backoffs = append(backoffs, delay)
	}

	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err, "Should create stderr pipe")
	originalStderr := os.Stderr
	os.Stderr = stderrWriter
	t.Cleanup(func() {
		os.Stderr = originalStderr
	})

	names, err := resolveForecastWorkflowsFromRemote([]string{"ci-doctor", "daily-planner"}, "owner/repo", true)
	require.NoError(t, err, "T-FC-030 should return caller-supplied partial results after rate-limit retries")
	assert.Equal(t, []string{"ci-doctor", "daily-planner"}, names, "Should preserve caller-supplied workflow order")
	assert.Equal(t, forecastRateLimitMaxAttempts, attempts, "Should retry discovery until the retry budget is exhausted")
	assert.Equal(t, []time.Duration{
		forecastRateLimitBackoffDuration(1),
		forecastRateLimitBackoffDuration(2),
	}, backoffs, "Should back off between retry attempts")

	require.NoError(t, stderrWriter.Close(), "Should close stderr writer after the test call")
	stderrBytes, readErr := io.ReadAll(stderrReader)
	require.NoError(t, readErr, "Should read captured stderr")
	assert.Contains(t, string(stderrBytes), "partial results", "Should warn that discovery returned partial results")
}
