//go:build !integration

package cli

// Formal model tests for the forecast compliance fixtures described in
// specs/forecast-compliance-fixtures/README.md.
//
// These tests encode predicates P1–P12 of the formal model over the fixture shape,
// the field mappings documented in the README's "Fixture Schema Reference" table,
// and the duration / Monte Carlo computations that consume those fixtures
// (pkg/cli/forecast_compute.go, pkg/cli/forecast_montecarlo.go).
//
//	P1  BernoulliSuccess      conclusion "success" maps to a Bernoulli success
//	P2  CancelledNotSuccess   cancelled/failure/timed_out are in-sample, not successes
//	P3  DurationNonNegative   duration derived from updatedAt/startedAt is >= 0
//	P4  ETObservationMapping  total_effective_tokens maps to the ET observation
//	P5  ZeroETExcludedFromAIC runs with AIC <= 0 are excluded from observations
//	P6  HighETNoOverflow      high AIC converts to milli-units without wraparound
//	P7  MonteCarloNilGuard    nil guards for empty/degenerate simulation inputs
//	P8  ReliabilityThreshold  IsReliable flips at the 10-observation boundary
//	P9  PercentileMonotonicity P10 <= P50 <= P90
//	P10 SuccessRateBounded    success rate stays within [0,1]
//	P11 PartialRunETValid     in-progress fixture ET snapshot is valid
//	P12 RunIDConsistency      top-level run_id matches nested run.id

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixtureRunSummary reads a fixture and decodes it into the RunSummary struct
// that the forecast command consumes at runtime.
func loadFixtureRunSummary(t *testing.T, name string) RunSummary {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureDir(t), name))
	require.NoError(t, err, "fixture file %q must be readable", name)
	var summary RunSummary
	require.NoError(t, json.Unmarshal(data, &summary), "fixture file %q must decode as RunSummary", name)
	return summary
}

// TestBernoulliSuccess_ConclusionMapping verifies predicate P1: a fixture whose
// run.conclusion is "success" is both in-sample and counted as a Bernoulli success.
func TestBernoulliSuccess_ConclusionMapping(t *testing.T) {
	for _, name := range []string{"run_summary_minimal.json", "run_summary_high_et.json", "run_summary_zero_et.json"} {
		t.Run(name, func(t *testing.T) {
			summary := loadFixtureRunSummary(t, name)
			require.Equal(t, "success", summary.Run.Conclusion,
				"P1: fixture %s must model a successful run", name)
			assert.True(t, isCompletedNonSkippedRun(summary.Run),
				"P1: successful run must be included in the Bernoulli sample")
			assert.True(t, isBernoulliSuccessRun(summary.Run),
				"P1: conclusion \"success\" must map to a Bernoulli success")
		})
	}
}

// TestCancelledFailureConclusions_ExcludedFromSuccess verifies predicate P2:
// cancelled, failed, and timed-out runs stay in the sample but are not successes.
func TestCancelledFailureConclusions_ExcludedFromSuccess(t *testing.T) {
	for _, name := range []string{"run_summary_cancelled.json", "run_summary_failed.json"} {
		t.Run(name, func(t *testing.T) {
			summary := loadFixtureRunSummary(t, name)
			assert.True(t, isCompletedNonSkippedRun(summary.Run),
				"P2: %s must remain in the sample (status completed, not skipped)", name)
			assert.False(t, isBernoulliSuccessRun(summary.Run),
				"P2: conclusion %q must not count as a Bernoulli success", summary.Run.Conclusion)
		})
	}

	// The same rule applies to every non-success terminal conclusion.
	for _, conclusion := range []string{"failure", "cancelled", "timed_out", "action_required", "neutral"} {
		run := WorkflowRun{Status: "completed", Conclusion: conclusion}
		assert.True(t, isCompletedNonSkippedRun(run),
			"P2: %q run must remain in the sample", conclusion)
		assert.False(t, isBernoulliSuccessRun(run),
			"P2: %q must not count as a Bernoulli success", conclusion)
	}

	// Skipped runs are excluded from the sample entirely.
	assert.False(t, isCompletedNonSkippedRun(WorkflowRun{Status: "completed", Conclusion: "skipped"}),
		"P2: skipped runs must be excluded from the sample")
}

// TestDurationSeconds_NonNegative verifies predicate P3: durations derived from
// run.updatedAt / run.startedAt are never negative.
func TestDurationSeconds_NonNegative(t *testing.T) {
	for _, name := range documentedForecastFixtures {
		t.Run(name, func(t *testing.T) {
			summary := loadFixtureRunSummary(t, name)
			require.False(t, summary.Run.StartedAt.IsZero(), "P3: %s must have run.startedAt", name)
			require.False(t, summary.Run.UpdatedAt.IsZero(), "P3: %s must have run.updatedAt", name)

			seconds := forecastRunDuration(summary.Run).Seconds()
			assert.GreaterOrEqual(t, seconds, 0.0,
				"P3: duration derived from %s must be non-negative", name)
			assert.InDelta(t, math.Max(0, summary.Run.UpdatedAt.Sub(summary.Run.StartedAt).Seconds()), seconds, 1e-9,
				"P3: duration must equal max(0, updatedAt - startedAt) for %s", name)
		})
	}

	// Inconsistent timestamps must not yield a negative duration.
	inverted := WorkflowRun{
		Status:     "completed",
		Conclusion: "success",
		StartedAt:  time.Date(2026, 5, 1, 11, 5, 30, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 5, 1, 11, 0, 5, 0, time.UTC),
	}
	assert.GreaterOrEqual(t, forecastRunDuration(inverted).Seconds(), 0.0,
		"P3: inverted timestamps must collapse to a non-negative duration")

	// Missing timestamps yield a zero duration rather than a garbage value.
	assert.Equal(t, time.Duration(0), forecastRunDuration(WorkflowRun{Status: "completed"}),
		"P3: missing timestamps must yield a zero duration")
}

// TestEffectiveTokensObservationMapping verifies predicate P4: the JSON field
// token_usage_summary.total_effective_tokens maps directly onto
// RunSummary.TokenUsage.TotalEffectiveTokens, the ET observation used by the
// bootstrap sampler.
func TestEffectiveTokensObservationMapping(t *testing.T) {
	for _, name := range documentedForecastFixtures {
		t.Run(name, func(t *testing.T) {
			raw := loadFixture(t, name)
			usage, ok := raw["token_usage_summary"].(map[string]any)
			require.True(t, ok, "P4: %s must contain a token_usage_summary object", name)

			summary := loadFixtureRunSummary(t, name)
			require.NotNil(t, summary.TokenUsage, "P4: %s must decode token_usage_summary", name)

			// The zero-ET fixture omits the value via omitempty semantics or sets 0.
			wantET := 0.0
			if v, present := usage["total_effective_tokens"]; present {
				wantET, ok = v.(float64)
				require.True(t, ok, "P4: total_effective_tokens must be a number in %s", name)
			}
			assert.InDelta(t, wantET, float64(summary.TokenUsage.TotalEffectiveTokens), 0.0,
				"P4: total_effective_tokens must map to TokenUsage.TotalEffectiveTokens in %s", name)
			assert.GreaterOrEqual(t, summary.TokenUsage.TotalEffectiveTokens, 0,
				"P4: ET observations must be non-negative in %s", name)
		})
	}
}

// TestZeroOrMissingAIC_ExcludedFromObservations verifies predicate P5: runs whose
// AIC is zero, negative, or non-finite are excluded from the Monte Carlo bootstrap
// observations.
func TestZeroOrMissingAIC_ExcludedFromObservations(t *testing.T) {
	cases := []struct {
		name     string
		aic      float64
		usable   bool
		wantMill int
	}{
		{name: "zero AIC (artifact missing)", aic: 0, usable: false},
		{name: "negative AIC", aic: -1.5, usable: false},
		{name: "NaN AIC", aic: math.NaN(), usable: false},
		{name: "+Inf AIC", aic: math.Inf(1), usable: false},
		{name: "-Inf AIC", aic: math.Inf(-1), usable: false},
		{name: "sub-unit AIC", aic: 0.0054, usable: true, wantMill: 5},
		{name: "unit AIC", aic: 1.0, usable: true, wantMill: 1000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			milli, usable := forecastAICObservation(tc.aic)
			assert.Equal(t, tc.usable, usable, "P5: usability mismatch for AIC=%v", tc.aic)
			if tc.usable {
				assert.Equal(t, tc.wantMill, milli, "P5: milli-AIC mismatch for AIC=%v", tc.aic)
			} else {
				assert.Equal(t, 0, milli, "P5: excluded observations must contribute zero milli-AIC")
			}
		})
	}

	// The cancelled fixture models a run that produced no AIC, so it is excluded.
	cancelled := loadFixtureRunSummary(t, "run_summary_cancelled.json")
	require.NotNil(t, cancelled.TokenUsage, "P5: cancelled fixture must decode token_usage_summary")
	_, usable := forecastAICObservation(cancelled.TokenUsage.TotalAIC)
	assert.False(t, usable,
		"P5: run_summary_cancelled.json has AIC <= 0 and must be excluded from observations")

	// The minimal fixture carries a positive AIC and is included.
	minimal := loadFixtureRunSummary(t, "run_summary_minimal.json")
	require.NotNil(t, minimal.TokenUsage, "P5: minimal fixture must decode token_usage_summary")
	milli, usable := forecastAICObservation(minimal.TokenUsage.TotalAIC)
	assert.True(t, usable, "P5: run_summary_minimal.json must contribute an observation")
	assert.Positive(t, milli, "P5: included observations must be positive milli-AIC")
}

// TestHighAIC_MilliConversionNoOverflow verifies predicate P6: very large AIC
// values convert to milli-units without overflowing into negative observations.
func TestHighAIC_MilliConversionNoOverflow(t *testing.T) {
	highET := loadFixtureRunSummary(t, "run_summary_high_et.json")
	require.NotNil(t, highET.TokenUsage, "P6: high-ET fixture must decode token_usage_summary")
	assert.GreaterOrEqual(t, highET.TokenUsage.TotalEffectiveTokens, 1_000_000,
		"P6: high-ET fixture must model the overflow boundary")

	milli, usable := forecastAICObservation(highET.TokenUsage.TotalAIC)
	require.True(t, usable, "P6: high-ET fixture must contribute an observation")
	assert.Positive(t, milli, "P6: high-ET observation must remain positive")

	for _, aic := range []float64{1e6, 1e12, 1e15, 1e18, 1e300, math.MaxFloat64} {
		milli, usable := forecastAICObservation(aic)
		require.True(t, usable, "P6: finite positive AIC %v must be usable", aic)
		assert.Positive(t, milli,
			"P6: AIC %v must not wrap around to a negative milli-AIC observation", aic)
	}
}

// TestRunMonteCarlo_NilGuardConditions verifies predicate P7: the simulation
// returns nil for empty observations and for degenerate run-rate inputs.
func TestRunMonteCarlo_NilGuardConditions(t *testing.T) {
	observations := []int{5_000, 6_000, 7_000}

	cases := []struct {
		name          string
		observations  []int
		successCount  int
		runsPerPeriod float64
		wantNil       bool
	}{
		{name: "nil observations", observations: nil, runsPerPeriod: 5, wantNil: true},
		{name: "empty observations", observations: []int{}, runsPerPeriod: 5, wantNil: true},
		{name: "zero runs per period", observations: observations, successCount: 3, runsPerPeriod: 0, wantNil: true},
		{name: "negative runs per period", observations: observations, successCount: 3, runsPerPeriod: -1, wantNil: true},
		{name: "NaN runs per period", observations: observations, successCount: 3, runsPerPeriod: math.NaN(), wantNil: true},
		{name: "+Inf runs per period", observations: observations, successCount: 3, runsPerPeriod: math.Inf(1), wantNil: true},
		{name: "-Inf runs per period", observations: observations, successCount: 3, runsPerPeriod: math.Inf(-1), wantNil: true},
		{name: "valid inputs", observations: observations, successCount: 3, runsPerPeriod: 5, wantNil: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test RNG
			got := runMonteCarlo(tc.observations, tc.successCount, tc.runsPerPeriod, rng)
			if tc.wantNil {
				assert.Nil(t, got, "P7: %s must produce a nil summary", tc.name)
				return
			}
			require.NotNil(t, got, "P7: %s must produce a summary", tc.name)
			assert.Equal(t, monteCarloIterations, got.Iterations,
				"P7: a valid simulation must report the configured iteration count")
		})
	}
}

// TestMonteCarloSummary_ReliabilityThreshold verifies predicate P8: IsReliable
// flips at the minObservationsForReliableForecast (10) observation boundary.
func TestMonteCarloSummary_ReliabilityThreshold(t *testing.T) {
	makeObservations := func(n int) []int {
		obs := make([]int, n)
		for i := range obs {
			obs[i] = 5_000 + i*100
		}
		return obs
	}

	for _, n := range []int{1, 5, 9, 10, 11, 25} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			rng := rand.New(rand.NewSource(7)) //nolint:gosec // deterministic test RNG
			summary := runMonteCarlo(makeObservations(n), n, 5, rng)
			require.NotNil(t, summary, "P8: %d observations must produce a summary", n)
			assert.Equal(t, n >= minObservationsForReliableForecast, summary.IsReliable,
				"P8: IsReliable must be true exactly when observations >= %d (n=%d)",
				minObservationsForReliableForecast, n)
		})
	}
}

// TestMonteCarloSummary_PercentileMonotonicity verifies predicate P9:
// P10 <= P50 <= P90 for every simulated summary.
func TestMonteCarloSummary_PercentileMonotonicity(t *testing.T) {
	cases := []struct {
		name          string
		observations  []int
		successCount  int
		runsPerPeriod float64
	}{
		{name: "small sample", observations: []int{1_000, 2_000, 3_000}, successCount: 2, runsPerPeriod: 4},
		{name: "large lambda (normal approximation)", observations: []int{5_400, 6_100, 7_200, 8_000}, successCount: 4, runsPerPeriod: 60},
		{name: "zero success rate", observations: []int{5_400, 6_100}, successCount: 0, runsPerPeriod: 10},
		{name: "high AIC observations", observations: []int{1_000_000, 2_000_000}, successCount: 2, runsPerPeriod: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(99)) //nolint:gosec // deterministic test RNG
			summary := runMonteCarlo(tc.observations, tc.successCount, tc.runsPerPeriod, rng)
			require.NotNil(t, summary, "P9: %s must produce a summary", tc.name)

			assert.LessOrEqual(t, summary.P10ProjectedAIC, summary.P50ProjectedAIC,
				"P9: P10 must be <= P50 for %s", tc.name)
			assert.LessOrEqual(t, summary.P50ProjectedAIC, summary.P90ProjectedAIC,
				"P9: P50 must be <= P90 for %s", tc.name)
			assert.GreaterOrEqual(t, summary.P10ProjectedAIC, 0.0,
				"P9: percentiles must be non-negative for %s", tc.name)
			assert.GreaterOrEqual(t, summary.StdDevAIC, 0.0,
				"P9: standard deviation must be non-negative for %s", tc.name)
		})
	}
}

// TestSuccessRate_BoundedZeroToOne verifies predicate P10: the Bernoulli success
// rate always lies within [0,1], including for degenerate counts.
func TestSuccessRate_BoundedZeroToOne(t *testing.T) {
	cases := []struct {
		name         string
		successCount int
		n            int
		want         float64
	}{
		{name: "no runs", successCount: 0, n: 0, want: 0},
		{name: "negative sample size", successCount: 3, n: -1, want: 0},
		{name: "no successes", successCount: 0, n: 10, want: 0},
		{name: "half successes", successCount: 5, n: 10, want: 0.5},
		{name: "all successes", successCount: 10, n: 10, want: 1},
		{name: "success count above sample size", successCount: 12, n: 10, want: 1},
		{name: "negative success count", successCount: -3, n: 10, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rate := forecastSuccessRate(tc.successCount, tc.n)
			assert.False(t, math.IsNaN(rate), "P10: success rate must never be NaN (%s)", tc.name)
			assert.GreaterOrEqual(t, rate, 0.0, "P10: success rate must be >= 0 (%s)", tc.name)
			assert.LessOrEqual(t, rate, 1.0, "P10: success rate must be <= 1 (%s)", tc.name)
			assert.InDelta(t, tc.want, rate, 1e-9, "P10: unexpected success rate for %s", tc.name)
		})
	}
}

// TestInProgressRunFixture_NonZeroETSnapshot verifies predicate P11: the
// in-progress fixture carries a valid, non-negative ET snapshot and is excluded
// from the completed-run sample.
func TestInProgressRunFixture_NonZeroETSnapshot(t *testing.T) {
	summary := loadFixtureRunSummary(t, "run_summary_partial_et.json")

	assert.Equal(t, "in_progress", summary.Run.Status,
		"P11: partial fixture must model an in-progress run")
	assert.False(t, isCompletedNonSkippedRun(summary.Run),
		"P11: in-progress runs must be excluded from the completed-run sample")

	require.NotNil(t, summary.TokenUsage, "P11: partial fixture must decode token_usage_summary")
	assert.Positive(t, summary.TokenUsage.TotalEffectiveTokens,
		"P11: in-progress fixture must carry a non-zero ET snapshot")
	assert.Positive(t, summary.TokenUsage.TotalAIC,
		"P11: in-progress fixture must carry a positive AIC snapshot")
	assert.GreaterOrEqual(t, forecastRunDuration(summary.Run).Seconds(), 0.0,
		"P11: in-progress fixture duration must be non-negative")
}

// TestFixture_RunIDConsistency verifies predicate P12: the top-level run_id equals
// the nested run.id in every fixture.
func TestFixture_RunIDConsistency(t *testing.T) {
	for _, name := range documentedForecastFixtures {
		t.Run(name, func(t *testing.T) {
			fixture := loadFixture(t, name)

			topLevel, ok := fixture["run_id"].(float64)
			require.True(t, ok, "P12: run_id must be a number in %s", name)
			assert.NotEqual(t, 0.0, topLevel, "P12: run_id must be non-zero in %s", name)

			run, ok := fixture["run"].(map[string]any)
			require.True(t, ok, "P12: 'run' must be a JSON object in %s", name)
			nested, ok := run["id"].(float64)
			require.True(t, ok, "P12: run.id must be a number in %s", name)

			assert.InDelta(t, topLevel, nested, 0.0,
				"P12: top-level run_id must match run.id in %s", name)

			summary := loadFixtureRunSummary(t, name)
			assert.Equal(t, int64(topLevel), summary.RunID,
				"P12: run_id must decode into RunSummary.RunID in %s", name)
		})
	}
}
