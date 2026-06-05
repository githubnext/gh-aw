package cli

// This file implements a Monte Carlo simulation engine for the forecast command.
// It models three independent sources of uncertainty:
//
//  1. Run-count uncertainty — the number of workflow executions in a future period
//     follows a Poisson process.  The arrival rate λ is itself uncertain (estimated
//     from a finite history window), so each trial draws λ from its Bayesian posterior
//     Gamma(n+0.5, scale=observedRunsPerPeriod/n), where n is the observed run count
//     and 0.5 is the Jeffreys non-informative prior shape.  This Gamma–Poisson
//     (Negative Binomial) compound model naturally produces wider confidence intervals
//     when data are sparse and converges to the classical Poisson estimate as n grows.
//  2. Per-run AIC variability — AIC per run is drawn via
//     bootstrap resampling from the historical observations, capturing the empirical
//     distribution without assuming a parametric form.
//  3. Per-run success uncertainty — each run independently succeeds with probability
//     equal to the historical success rate (Bernoulli model).
//
// Running 10 000 trials and reporting P10/P50/P90 gives conservative and optimistic
// estimates alongside the median, which is more informative than a single point
// estimate for capacity planning.

import (
	"math"
	"math/rand"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var forecastMonteCarloLog = logger.New("cli:forecast_montecarlo")

// monteCarloIterations is the number of simulation trials per workflow.
// 10 000 gives < 1% Monte Carlo error on percentile estimates and runs in < 10 ms
// for typical sample sizes.
const monteCarloIterations = 10_000

// poissonNormalApproximationThreshold is the normative λ crossover threshold:
// Knuth's exact algorithm is used for λ ≤ threshold, and Normal approximation
// is used only for λ > threshold.
//
// Specification reference: forecast-specification.md Appendix B ("Poisson Algorithm
// Selection Rationale") documents the error analysis justifying λ = 15 as the crossover
// point. Re-review Appendix B whenever this threshold is changed, per R-FC-060 and
// open issue #31985.
const poissonNormalApproximationThreshold = 15.0

// minObservationsForReliableForecast is the minimum number of completed run
// observations required for confidence intervals to be considered statistically
// meaningful.  Forecasts based on fewer observations are returned but flagged
// IsReliable = false.
const minObservationsForReliableForecast = 10

// ForecastMonteCarloSummary contains the probability distribution of projected
// AIC totals derived from a Monte Carlo simulation.
//
// The simulation models run-count uncertainty via a Gamma–Poisson (Negative
// Binomial) compound process, per-run token usage via bootstrap resampling of
// historical observations, and per-run success probability via a Bernoulli draw.
// Percentile estimates (P10/P50/P90) give optimistic, median, and conservative
// bounds for the forecast period.
type ForecastMonteCarloSummary struct {
	// Iterations is the number of simulation trials that were run.
	Iterations int `json:"iterations"`
	// MeanProjectedAIC is the arithmetic mean of simulated AIC totals across all trials.
	MeanProjectedAIC float64 `json:"mean_projected_aic"`
	// StdDevAIC is the standard deviation of simulated AIC totals (spread of distribution).
	StdDevAIC float64 `json:"std_dev_aic"`
	// P10ProjectedAIC is the 10th-percentile AIC total — only 10% of simulated outcomes
	// fall below this value (optimistic bound).
	P10ProjectedAIC float64 `json:"p10_projected_aic"`
	// P50ProjectedAIC is the median simulated AIC total.
	P50ProjectedAIC float64 `json:"p50_projected_aic"`
	// P90ProjectedAIC is the 90th-percentile AIC total — 90% of simulated outcomes fall
	// below this value (conservative / budget bound).
	P90ProjectedAIC float64 `json:"p90_projected_aic"`
	// IsReliable is true when the simulation was based on at least minObservationsForReliableForecast
	// completed runs.  When false the confidence intervals may be very wide or unreliable.
	IsReliable bool `json:"is_reliable"`
}

// runMonteCarlo runs a Monte Carlo simulation to estimate the probability distribution
// of projected AIC usage over the forecast period.
//
// Parameters:
//   - aicObservations: per-run AIC values represented in milli-AIC units from
//     historical completed runs.
//   - successCount: number of those runs that concluded "success".
//   - observedRunsPerPeriod: point estimate of expected runs in the projection period.
//   - rng: caller-supplied random number generator (allows deterministic testing).
//
// The run-count rate λ is treated as uncertain and drawn each trial from its
// Bayesian posterior Gamma(n+0.5, scale=observedRunsPerPeriod/n), where n is the
// number of historical observations and 0.5 is the Jeffreys non-informative prior
// shape.  This compound Gamma–Poisson model is equivalent to a Negative Binomial
// and naturally produces wider confidence intervals for small samples, converging to
// the classical Poisson(observedRunsPerPeriod) model as n → ∞.
//
// Returns nil when aicObservations is empty or observedRunsPerPeriod ≤ 0.
func runMonteCarlo(aicObservations []int, successCount int, observedRunsPerPeriod float64, rng *rand.Rand) *ForecastMonteCarloSummary {
	n := len(aicObservations)
	if n == 0 || observedRunsPerPeriod <= 0 || math.IsNaN(observedRunsPerPeriod) || math.IsInf(observedRunsPerPeriod, 0) {
		forecastMonteCarloLog.Printf("Skipping Monte Carlo: observations=%d, runs_per_period=%.2f", n, observedRunsPerPeriod)
		return nil
	}

	successRate := float64(successCount) / float64(n)
	forecastMonteCarloLog.Printf("Running Monte Carlo: observations=%d, success_count=%d, success_rate=%.3f, runs_per_period=%.2f, iterations=%d",
		n, successCount, successRate, observedRunsPerPeriod, monteCarloIterations)

	// Bayesian posterior parameters for the Poisson arrival rate λ.
	// Prior: Jeffreys improper prior ∝ 1/√λ — equivalent to Gamma(0.5, ∞).
	// Likelihood: observedCount ~ Poisson(λ × historyWindow).
	// Posterior: λ_period | n ~ Gamma(shape=n+0.5, scale=observedRunsPerPeriod/n).
	// Mean of this Gamma = (n+0.5)/n × observedRunsPerPeriod ≈ observedRunsPerPeriod.
	gammaShape := float64(n) + 0.5
	gammaScale := observedRunsPerPeriod / float64(n)

	simAICMilli := make([]int, monteCarloIterations)

	for i := range monteCarloIterations {
		// Draw run-count rate from posterior Gamma (accounts for estimation uncertainty in λ).
		lambdaTrial := gammaSample(rng, gammaShape) * gammaScale
		// Draw number of runs from Poisson(λ_trial).
		numRuns := poissonSample(rng, lambdaTrial)

		var totalAICMilli int
		for range numRuns {
			// Each run succeeds independently with probability successRate.
			if rng.Float64() >= successRate {
				continue
			}
			// Bootstrap: sample AIC from the empirical distribution.
			totalAICMilli += aicObservations[rng.Intn(n)]
		}

		simAICMilli[i] = totalAICMilli
	}

	// Sort for percentile computation.
	sort.Ints(simAICMilli)

	meanMilli, stddevMilli := meanStdDevInt(simAICMilli)
	mean := roundForecastAIC(float64(meanMilli) / 1000)
	stddev := roundForecastAIC(stddevMilli / 1000)
	p10 := roundForecastAIC(float64(percentileInt(simAICMilli, 10)) / 1000)
	p50 := roundForecastAIC(float64(percentileInt(simAICMilli, 50)) / 1000)
	p90 := roundForecastAIC(float64(percentileInt(simAICMilli, 90)) / 1000)
	reliable := n >= minObservationsForReliableForecast
	forecastMonteCarloLog.Printf("Monte Carlo complete: mean_aic=%.3f, stddev=%.3f, p10=%.3f, p50=%.3f, p90=%.3f, reliable=%v",
		mean, stddev, p10, p50, p90, reliable)

	return &ForecastMonteCarloSummary{
		Iterations:       monteCarloIterations,
		MeanProjectedAIC: mean,
		StdDevAIC:        stddev,
		P10ProjectedAIC:  p10,
		P50ProjectedAIC:  p50,
		P90ProjectedAIC:  p90,
		IsReliable:       reliable,
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
	if !useNormalApproximationForPoisson(lambda) {
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

func useNormalApproximationForPoisson(lambda float64) bool {
	return lambda > poissonNormalApproximationThreshold
}

// gammaSample draws a random variate from Gamma(shape, scale=1) using the
// Marsaglia-Tsang squeeze method for shape ≥ 1, and the reduction
// Gamma(shape) = Gamma(shape+1) × U^(1/shape) for 0 < shape < 1.
//
// References: Marsaglia & Tsang (2000), "A Simple Method for Generating Gamma Variables".
//
// shape ≤ 0 is a caller error; the function returns 0 as a defensive no-op
// consistent with poissonSample's treatment of lambda ≤ 0.  All call sites in the
// simulation pass shape = n+0.5 (n ≥ 1), so this branch is never reached in
// practice.
func gammaSample(rng *rand.Rand, shape float64) float64 {
	if shape <= 0 {
		return 0
	}
	if shape < 1 {
		// Reduce to shape+1 via the identity X = Y × U^(1/shape).
		return gammaSample(rng, shape+1) * math.Pow(rng.Float64(), 1.0/shape)
	}
	// Marsaglia-Tsang method for shape ≥ 1.
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		var x, v float64
		for {
			x = rng.NormFloat64()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := rng.Float64()
		xsq := x * x
		// Fast acceptance (squeeze step).
		if u < 1.0-0.0331*(xsq*xsq) {
			return d * v
		}
		// Slower acceptance (log-space step).
		if math.Log(u) < 0.5*xsq+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

// meanStdDevInt computes the arithmetic mean and population standard deviation
// of the int slice xs (assumed non-empty).
//
// The mean is returned as an int (truncated toward zero after integer division),
// which is used for the milli-AIC intermediate representation.
// The standard deviation uses the full floating-point mean to avoid accumulating
// rounding error in the variance calculation.
func meanStdDevInt(xs []int) (mean int, stddev float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum int
	for _, x := range xs {
		sum += x
	}
	mean = sum / len(xs)
	// Use the exact float mean for stddev to avoid bias from integer truncation.
	fmean := float64(sum) / float64(len(xs))
	for _, x := range xs {
		d := float64(x) - fmean
		stddev += d * d
	}
	stddev = math.Sqrt(stddev / float64(len(xs)))
	return
}

// percentileInt returns the p-th percentile of an already-sorted int slice
// using the nearest-rank method.  p must be in [1, 100].
func percentileInt(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	idx = max(idx, 0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
