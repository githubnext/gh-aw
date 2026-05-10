package cli

// This file implements a Monte Carlo simulation engine for the forecast command.
// It models three independent sources of uncertainty:
//
//  1. Run-count uncertainty — the number of workflow executions in a future period
//     follows a Poisson process with rate λ = observed runs per period.
//  2. Per-run token usage variability — effective tokens per run are drawn via
//     bootstrap resampling from the historical observations, capturing the empirical
//     distribution without assuming a parametric form.
//  3. Per-run success uncertainty — each run independently succeeds with probability
//     equal to the historical success rate (Bernoulli model).
//
// Running 10 000 trials and reporting P10/P50/P90 gives conservative and optimistic
// estimates alongside the median, which is more informative than a single point
// estimate for capacity planning and cost budgeting.

import (
	"math"
	"math/rand"
	"sort"
)

// monteCarloIterations is the number of simulation trials per workflow.
// 10 000 gives < 1% Monte Carlo error on percentile estimates and runs in < 10 ms
// for typical sample sizes.
const monteCarloIterations = 10_000

// ForecastMonteCarloSummary contains the probability distribution of projected costs
// and effective-token counts derived from a Monte Carlo simulation.
//
// The simulation models run-count uncertainty via a Poisson process, per-run token
// usage via bootstrap resampling of historical observations, and per-run success
// probability via a Bernoulli draw.  Percentile estimates (P10/P50/P90) give
// optimistic, median, and conservative bounds for the forecast period.
type ForecastMonteCarloSummary struct {
	// Iterations is the number of simulation trials that were run.
	Iterations int `json:"iterations"`
	// MeanProjectedCostUSD is the arithmetic mean of simulated costs across all trials.
	MeanProjectedCostUSD float64 `json:"mean_projected_cost_usd"`
	// StdDevCostUSD is the standard deviation of simulated costs (spread of the distribution).
	StdDevCostUSD float64 `json:"std_dev_cost_usd"`
	// P10ProjectedCostUSD is the 10th-percentile cost — only 10% of simulated outcomes
	// fall below this value (optimistic bound).
	P10ProjectedCostUSD float64 `json:"p10_projected_cost_usd"`
	// P50ProjectedCostUSD is the median simulated cost.
	P50ProjectedCostUSD float64 `json:"p50_projected_cost_usd"`
	// P90ProjectedCostUSD is the 90th-percentile cost — 90% of simulated outcomes fall
	// below this value (conservative / budget bound).
	P90ProjectedCostUSD float64 `json:"p90_projected_cost_usd"`
	// P10ProjectedEffectiveTokens is the 10th-percentile effective-token count.
	P10ProjectedEffectiveTokens int `json:"p10_projected_effective_tokens"`
	// P50ProjectedEffectiveTokens is the median effective-token count.
	P50ProjectedEffectiveTokens int `json:"p50_projected_effective_tokens"`
	// P90ProjectedEffectiveTokens is the 90th-percentile effective-token count.
	P90ProjectedEffectiveTokens int `json:"p90_projected_effective_tokens"`
}

// runMonteCarlo runs a Monte Carlo simulation to estimate the probability distribution
// of projected effective-token usage and cost over the forecast period.
//
// Parameters:
//   - etObservations: per-run effective-token counts from historical completed runs.
//   - successCount: number of those runs that concluded "success".
//   - observedRunsPerPeriod: expected number of runs in the projection period (λ).
//   - rng: caller-supplied random number generator (allows deterministic testing).
//
// Returns nil when etObservations is empty or observedRunsPerPeriod ≤ 0.
func runMonteCarlo(etObservations []int, successCount int, observedRunsPerPeriod float64, rng *rand.Rand) *ForecastMonteCarloSummary {
	n := len(etObservations)
	if n == 0 || observedRunsPerPeriod <= 0 {
		return nil
	}

	successRate := float64(successCount) / float64(n)

	simCosts := make([]float64, monteCarloIterations)
	simETs := make([]int, monteCarloIterations)

	for i := 0; i < monteCarloIterations; i++ {
		// Draw number of runs from Poisson(λ = observedRunsPerPeriod).
		numRuns := poissonSample(rng, observedRunsPerPeriod)

		var totalET int
		for j := 0; j < numRuns; j++ {
			// Each run succeeds independently with probability successRate.
			if rng.Float64() >= successRate {
				continue
			}
			// Bootstrap: sample ET from the empirical distribution.
			totalET += etObservations[rng.Intn(n)]
		}

		simETs[i] = totalET
		simCosts[i] = float64(totalET) * costPerEffectiveToken
	}

	// Sort for percentile computation.
	sort.Float64s(simCosts)
	sort.Ints(simETs)

	mean, stddev := costMeanStdDev(simCosts)

	return &ForecastMonteCarloSummary{
		Iterations:                  monteCarloIterations,
		MeanProjectedCostUSD:        mean,
		StdDevCostUSD:               stddev,
		P10ProjectedCostUSD:         percentileFloat64(simCosts, 10),
		P50ProjectedCostUSD:         percentileFloat64(simCosts, 50),
		P90ProjectedCostUSD:         percentileFloat64(simCosts, 90),
		P10ProjectedEffectiveTokens: percentileInt(simETs, 10),
		P50ProjectedEffectiveTokens: percentileInt(simETs, 50),
		P90ProjectedEffectiveTokens: percentileInt(simETs, 90),
	}
}

// poissonSample draws a random variate from Poisson(lambda).
//
// For lambda ≤ 15 it uses Knuth's multiplicative algorithm (exact, O(lambda) per sample).
// For lambda > 15 it uses a Normal approximation, which is accurate to
// within 0.3% for the tails that matter in forecasting contexts, and avoids
// the linear cost that becomes significant at 10 000 trials.
func poissonSample(rng *rand.Rand, lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	if lambda <= 15 {
		// Knuth's algorithm: O(lambda) per sample, exact.
		L := math.Exp(-lambda)
		k := 0
		p := 1.0
		for {
			k++
			p *= rng.Float64()
			if p <= L {
				break
			}
		}
		return k - 1
	}
	// Normal approximation: Poisson(λ) ≈ N(λ, √λ) for large λ.
	v := lambda + math.Sqrt(lambda)*rng.NormFloat64()
	if v < 0 {
		return 0
	}
	return int(math.Round(v))
}

// costMeanStdDev computes the arithmetic mean and population standard deviation
// of the slice xs (assumed non-empty).
func costMeanStdDev(xs []float64) (mean, stddev float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	for _, x := range xs {
		d := x - mean
		stddev += d * d
	}
	stddev = math.Sqrt(stddev / float64(len(xs)))
	return
}

// percentileFloat64 returns the p-th percentile of an already-sorted float64 slice
// using the nearest-rank method.  p must be in [1, 100].
func percentileFloat64(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// percentileInt returns the p-th percentile of an already-sorted int slice
// using the nearest-rank method.  p must be in [1, 100].
func percentileInt(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
