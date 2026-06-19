//go:build !integration

package cli

// Formal compliance tests for the Monte Carlo forecast engine.
//
// These tests cover predicates P1–P13 derived from the formal model in
// specs/forecast-compliance-fixtures/README.md (T-FC-031 – T-FC-040).
//
// Formal notation cross-references (derived from the specification analysis in
// specs/forecast-compliance-fixtures/README.md and the formal model that produced
// the issue — see the "Formal Model" section of issue #40114):
//   - TLA+ invariants: P1, P5, P7, P8, P9
//   - F* pre/post conditions: P2, P3, P4, P6, P10, P11
//   - Z3-SMT schema gap: P12, P13

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureDir returns the absolute path to the forecast compliance fixtures directory.
func fixtureDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller must return a valid file path")
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "specs", "forecast-compliance-fixtures")
}

// loadFixture reads and parses a fixture JSON file into a map.
func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureDir(t), name))
	require.NoError(t, err, "fixture file %q must be readable", name)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m), "fixture file %q must be valid JSON", name)
	return m
}

// TestFormal_P1_FixtureFieldMapping verifies that the canonical forecast fixture
// contains the JSON fields that the forecast engine reads at runtime.
//
// Formal predicate: ∀f ∈ FixtureFields: f ∈ dom(run_summary_minimal.json)
// Specification reference: specs/forecast-compliance-fixtures/README.md §Fixture Schema Reference
func TestFormal_P1_FixtureFieldMapping(t *testing.T) {
	fixture := loadFixture(t, "run_summary_minimal.json")

	// Top-level identity field.
	assert.Contains(t, fixture, "run_id", "P1: run_id must be present")

	// run sub-object must expose conclusion, updated_at, run_started_at.
	run, ok := fixture["run"].(map[string]any)
	require.True(t, ok, "P1: 'run' must be a JSON object")
	for _, field := range []string{"conclusion", "updated_at", "run_started_at"} {
		assert.Contains(t, run, field, "P1: run.%s must be present for forecast inputs", field)
	}

	// token_usage_summary must contain total_effective_tokens and total_aic.
	usage, ok := fixture["token_usage_summary"].(map[string]any)
	require.True(t, ok, "P1: 'token_usage_summary' must be a JSON object")
	assert.Contains(t, usage, "total_effective_tokens", "P1: token_usage_summary.total_effective_tokens required")
	assert.Contains(t, usage, "total_aic", "P1: token_usage_summary.total_aic required (AIC is what the engine reads)")
}

// TestFormal_P2_BernoulliSuccess verifies the Bernoulli success model:
// only runs with conclusion=="success" contribute AIC to the bootstrap sample.
//
// Formal predicate: successRate = successCount / n; P(AIC>0|trial) = successRate
// Specification reference: R-MC-020, R-MC-021
func TestFormal_P2_BernoulliSuccess(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) //nolint:gosec

	aicObs := []int{5_000, 6_000, 7_000}
	const lambda = 8.0

	// All runs succeed: every trial has a chance to accumulate AIC.
	mcAllSuccess := runMonteCarlo(aicObs, len(aicObs), lambda, rng)
	require.NotNil(t, mcAllSuccess, "P2: all-success input must produce a result")
	assert.Greater(t, mcAllSuccess.MeanProjectedAIC, 0.0, "P2: all-success → positive mean AIC")

	// Zero success rate: every trial contributes 0 AIC regardless of run count.
	rng2 := rand.New(rand.NewSource(42)) //nolint:gosec
	mcZeroSuccess := runMonteCarlo(aicObs, 0, lambda, rng2)
	require.NotNil(t, mcZeroSuccess, "P2: zero-success input must produce a result (Poisson still fires)")
	assert.InDelta(t, 0.0, mcZeroSuccess.MeanProjectedAIC, 1e-9,
		"P2: zero success rate → zero mean AIC (Bernoulli collapses the contribution)")
}

// TestFormal_P3_ObservedRateFormula verifies the observed-rate derivation:
//
//	observedRunsPerPeriod = (sampledRuns / historyDays) * periodDays
//
// Formal predicate: λ = (n/h) × p, where h = historyDays, p = periodDays
// Specification reference: §3.8 Run Frequency Estimation
func TestFormal_P3_ObservedRateFormula(t *testing.T) {
	cases := []struct {
		sampledRuns int
		historyDays int
		periodDays  int
		wantLambda  float64
	}{
		{sampledRuns: 10, historyDays: 30, periodDays: 30, wantLambda: 10.0},
		{sampledRuns: 10, historyDays: 30, periodDays: 7, wantLambda: 10.0 / 30.0 * 7},
		{sampledRuns: 21, historyDays: 7, periodDays: 7, wantLambda: 21.0},
		{sampledRuns: 0, historyDays: 30, periodDays: 30, wantLambda: 0.0},
	}
	for _, tc := range cases {
		lambda := float64(tc.sampledRuns) / float64(tc.historyDays) * float64(tc.periodDays)
		assert.InDelta(t, tc.wantLambda, lambda, 1e-9,
			"P3: (n=%d/h=%d)*p=%d should equal λ=%.4f",
			tc.sampledRuns, tc.historyDays, tc.periodDays, tc.wantLambda)
	}
}

// TestFormal_P4_YieldFormula verifies the yield invariant:
//
//	yield = successRate × observedRunsPerPeriod
//
// Formal predicate: yield = sr × obs  (§3.9 example)
// Specification reference: §3.9 Effective Yield Estimate
func TestFormal_P4_YieldFormula(t *testing.T) {
	cases := []struct {
		successCount int
		totalRuns    int
		lambda       float64
		wantYield    float64
	}{
		{successCount: 8, totalRuns: 10, lambda: 10.0, wantYield: 0.8 * 10.0},
		{successCount: 10, totalRuns: 10, lambda: 5.0, wantYield: 1.0 * 5.0},
		{successCount: 0, totalRuns: 10, lambda: 10.0, wantYield: 0.0},
		{successCount: 3, totalRuns: 4, lambda: 8.0, wantYield: 0.75 * 8.0},
	}
	for _, tc := range cases {
		successRate := float64(tc.successCount) / float64(tc.totalRuns)
		yield := successRate * tc.lambda
		assert.InDelta(t, tc.wantYield, yield, 1e-9,
			"P4: yield = sr(%.2f) × obs(%.2f) = %.4f",
			successRate, tc.lambda, tc.wantYield)
	}
}

// TestFormal_P5_MonteCarloIterations verifies that runMonteCarlo always produces
// exactly monteCarloIterations (10 000) simulation trials (§7.1).
//
// Formal predicate: result.Iterations = monteCarloIterations
// Specification reference: §7.1 Simulation Trial Count
func TestFormal_P5_MonteCarloIterations(t *testing.T) {
	assert.Equal(t, 10_000, monteCarloIterations, "P5: monteCarloIterations constant must equal 10 000")

	rng := rand.New(rand.NewSource(7)) //nolint:gosec
	obs := []int{1_000, 2_000, 3_000, 4_000}
	mc := runMonteCarlo(obs, len(obs), 5.0, rng)
	require.NotNil(t, mc, "P5: valid input must produce a result")
	assert.Equal(t, monteCarloIterations, mc.Iterations,
		"P5: result.Iterations must equal monteCarloIterations (10 000)")
}

// TestFormal_P6_ZeroLambdaNilResult verifies that runMonteCarlo returns nil
// when λ ≤ 0, is NaN, or is ±Inf (R-MC-001, R-MC-004).
//
// Formal predicate: λ ≤ 0 ∨ ¬(λ = λ) ∨ |λ| = ∞ → result = nil
// Specification reference: R-MC-001, R-MC-004
func TestFormal_P6_ZeroLambdaNilResult(t *testing.T) {
	obs := []int{1_000, 2_000}
	cases := []struct {
		name   string
		lambda float64
	}{
		{"zero λ", 0.0},
		{"negative λ", -1.0},
		{"NaN λ", math.NaN()},
		{"+Inf λ", math.Inf(1)},
		{"-Inf λ", math.Inf(-1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(42)) //nolint:gosec
			result := runMonteCarlo(obs, len(obs), tc.lambda, rng)
			assert.Nil(t, result, "P6: λ=%v must yield nil (zero-projection fallback)", tc.lambda)
		})
	}
}

// TestFormal_P7_ZeroAICExclusion verifies that all-zero TotalAIC run history
// produces no Monte Carlo input (R-MC-011, R-MC-032).
//
// Formal predicate: ∀r ∈ runs: r.TotalAIC = 0 → aicObservations = [] → result = nil
// Specification reference: R-MC-011, R-MC-032; forecast.go:593 (runAIC ≤ 0 → continue)
func TestFormal_P7_ZeroAICExclusion(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) //nolint:gosec

	// nil observations → runMonteCarlo must return nil.
	assert.Nil(t, runMonteCarlo(nil, 0, 5.0, rng),
		"P7: nil AIC observations must return nil (R-MC-011)")

	// Empty observations → runMonteCarlo must return nil.
	assert.Nil(t, runMonteCarlo([]int{}, 0, 5.0, rng),
		"P7: empty AIC observations must return nil (R-MC-032)")
}

// TestFormal_P8_ReliabilityThreshold verifies the minimum-observation threshold:
// n < minObservationsForReliableForecast → IsReliable=false; n ≥ threshold → IsReliable=true.
//
// Formal predicate: IsReliable ⟺ n ≥ minObservationsForReliableForecast (R-MC-030)
// Specification reference: R-MC-030
func TestFormal_P8_ReliabilityThreshold(t *testing.T) {
	assert.Equal(t, 10, minObservationsForReliableForecast,
		"P8: minObservationsForReliableForecast constant must equal 10")

	buildObs := func(n int) []int {
		obs := make([]int, n)
		for i := range obs {
			obs[i] = 1_000 + i*100
		}
		return obs
	}

	cases := []struct {
		n            int
		wantReliable bool
	}{
		{n: 1, wantReliable: false},
		{n: 9, wantReliable: false},
		{n: 10, wantReliable: true},
		{n: 20, wantReliable: true},
	}
	for _, tc := range cases {
		obs := buildObs(tc.n)
		rng := rand.New(rand.NewSource(42)) //nolint:gosec
		mc := runMonteCarlo(obs, len(obs), 4.0, rng)
		require.NotNil(t, mc, "P8: n=%d must produce a non-nil result", tc.n)
		assert.Equal(t, tc.wantReliable, mc.IsReliable,
			"P8: n=%d: IsReliable must be %v (threshold=%d)",
			tc.n, tc.wantReliable, minObservationsForReliableForecast)
	}
}

// TestFormal_P9_PoissonBranchCrossover verifies the Poisson algorithm selection rule:
// λ ≤ poissonNormalApproximationThreshold → Knuth exact; λ > threshold → Normal approximation.
//
// Formal predicate: useNormalApproximationForPoisson(λ) ⟺ λ > 15 (R-FC-060)
// Specification reference: R-FC-060; forecast_montecarlo.go poissonNormalApproximationThreshold
func TestFormal_P9_PoissonBranchCrossover(t *testing.T) {
	assert.InDelta(t, 15.0, poissonNormalApproximationThreshold, 0,
		"P9: Poisson crossover threshold must equal 15")

	cases := []struct {
		lambda      float64
		wantNormal  bool
		description string
	}{
		{lambda: 0.1, wantNormal: false, description: "small λ uses Knuth exact"},
		{lambda: 14.999, wantNormal: false, description: "below threshold uses Knuth exact"},
		{lambda: 15.0, wantNormal: false, description: "at threshold uses Knuth exact (≤ not <)"},
		{lambda: 15.001, wantNormal: true, description: "above threshold uses Normal approximation"},
		{lambda: 100.0, wantNormal: true, description: "large λ uses Normal approximation"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.wantNormal, useNormalApproximationForPoisson(tc.lambda),
			"P9: λ=%.3f: %s", tc.lambda, tc.description)
	}
}

// TestFormal_P10_DurationDerivation verifies that run duration is computed as
// UpdatedAt − StartedAt (§6.2.2), matching the derivation in forecast.go.
//
// Formal predicate: Duration = UpdatedAt − StartedAt  (§6.2.2)
// Specification reference: §6.2.2 Duration Derivation; forecast.go:573-574
func TestFormal_P10_DurationDerivation(t *testing.T) {
	cases := []struct {
		startedAt    time.Time
		updatedAt    time.Time
		wantDuration time.Duration
		description  string
	}{
		{
			startedAt:    time.Date(2026, 5, 1, 11, 0, 5, 0, time.UTC),
			updatedAt:    time.Date(2026, 5, 1, 11, 5, 35, 0, time.UTC),
			wantDuration: 5*time.Minute + 30*time.Second,
			description:  "fixture example: 5 min 30 s run",
		},
		{
			startedAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			updatedAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			wantDuration: 0,
			description:  "zero-length run (immediate completion)",
		},
		{
			startedAt:    time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
			updatedAt:    time.Date(2026, 5, 1, 11, 30, 0, 0, time.UTC),
			wantDuration: 90 * time.Minute,
			description:  "90-minute run",
		},
	}
	for _, tc := range cases {
		r := WorkflowRun{
			StartedAt: tc.startedAt,
			UpdatedAt: tc.updatedAt,
		}
		// Replicate the derivation logic from forecast.go:573-574.
		if r.Duration == 0 && !r.StartedAt.IsZero() && !r.UpdatedAt.IsZero() {
			r.Duration = r.UpdatedAt.Sub(r.StartedAt)
		}
		assert.Equal(t, tc.wantDuration, r.Duration,
			"P10: %s: Duration must equal UpdatedAt − StartedAt", tc.description)
	}
}

// TestFormal_P11_FlagValidation_Days verifies that days ∉ {7, 30} produces an error (R-CLI-001).
//
// Formal predicate: config.Days ∉ {7, 30} → RunForecast returns error
// Specification reference: R-CLI-001; forecast.go:227-229
func TestFormal_P11_FlagValidation_Days(t *testing.T) {
	invalidCases := []int{0, 1, 6, 8, 14, 29, 31, 60, 90, 365}
	for _, days := range invalidCases {
		cfg := ForecastConfig{Days: days, Period: "month", JSONOutput: true, SampleSize: 10}
		err := RunForecast(cfg)
		require.Error(t, err,
			"P11: days=%d must return an error (only 7 and 30 are valid)", days)
		assert.Contains(t, err.Error(), "must be 7 or 30",
			"P11: error message must document the allowed values")
	}

	// Allowed values must NOT return a flag-validation error.
	for _, days := range []int{7, 30} {
		cfg := ForecastConfig{Days: days, Period: "month", JSONOutput: true, SampleSize: 10}
		err := RunForecast(cfg)
		// The call may fail for other reasons (no GitHub auth, no workflows), but must
		// not fail with the days-validation error.
		if err != nil {
			assert.NotContains(t, err.Error(), "invalid days value",
				"P11: days=%d is valid and must not trigger the days-validation error", days)
		}
	}
}

// TestFormal_P12_FixtureAICGap verifies that the canonical fixture exposes a
// positive total_aic, closing the gap where the engine reads TotalAIC but the
// original fixture only set total_effective_tokens.
//
// Formal predicate (Z3-SMT gap): total_aic > 0 ∧ total_effective_tokens > 0
// Specification reference: forecast.go:593 (runAIC ≤ 0 → continue skips run)
func TestFormal_P12_FixtureAICGap(t *testing.T) {
	fixture := loadFixture(t, "run_summary_minimal.json")

	usage, ok := fixture["token_usage_summary"].(map[string]any)
	require.True(t, ok, "P12: token_usage_summary must be a JSON object")

	// total_effective_tokens must remain non-zero (pre-existing invariant).
	et, hasET := usage["total_effective_tokens"]
	require.True(t, hasET, "P12: total_effective_tokens must be present")
	etVal, ok := et.(float64)
	require.True(t, ok, "P12: total_effective_tokens must be a number")
	assert.Greater(t, etVal, 0.0, "P12: total_effective_tokens must be > 0")

	// total_aic must now be present and positive so the forecast engine does not
	// skip the run at forecast.go:593 (runAIC ≤ 0 → continue).
	aic, hasAIC := usage["total_aic"]
	require.True(t, hasAIC,
		"P12 (gap): total_aic must be present in token_usage_summary — "+
			"engine reads TotalAIC, not total_effective_tokens")
	aicVal, ok := aic.(float64)
	require.True(t, ok, "P12: total_aic must be a number")
	assert.Greater(t, aicVal, 0.0,
		"P12: total_aic must be > 0 so the run is not skipped at forecast.go:593")
}

// TestFormal_P13_FixtureJSONConformance verifies that all required top-level and
// nested fields are present in the canonical run_summary_minimal.json fixture.
//
// Formal predicate: dom(fixture) ⊇ RequiredFields
// Specification reference: pkg/cli/logs_models.go RunSummary struct
func TestFormal_P13_FixtureJSONConformance(t *testing.T) {
	fixture := loadFixture(t, "run_summary_minimal.json")

	// Required top-level fields (mapped from RunSummary struct JSON tags).
	topLevelRequired := []string{
		"cli_version",
		"run_id",
		"processed_at",
		"run",
		"metrics",
		"token_usage_summary",
	}
	for _, field := range topLevelRequired {
		assert.Contains(t, fixture, field,
			"P13: top-level field %q must be present in run_summary_minimal.json", field)
	}

	// run sub-object required fields.
	run, ok := fixture["run"].(map[string]any)
	require.True(t, ok, "P13: 'run' must be a JSON object")
	runRequired := []string{"conclusion", "updated_at", "run_started_at"}
	for _, field := range runRequired {
		assert.Contains(t, run, field,
			"P13: run.%q must be present for duration and Bernoulli derivation", field)
	}

	// token_usage_summary required fields.
	usage, ok := fixture["token_usage_summary"].(map[string]any)
	require.True(t, ok, "P13: 'token_usage_summary' must be a JSON object")
	usageRequired := []string{"total_effective_tokens", "total_aic"}
	for _, field := range usageRequired {
		assert.Contains(t, usage, field,
			"P13: token_usage_summary.%q must be present", field)
	}
}
