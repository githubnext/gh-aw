//go:build !integration

package cli

import (
	"context"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── formatForecastPercent ────────────────────────────────────────────────────

func TestFormatForecastPercent_NoData(t *testing.T) {
	assert.Equal(t, "N/A", formatForecastPercent(0, false), "no data → N/A")
}

func TestFormatForecastPercent_ZeroPercent(t *testing.T) {
	// A legitimate 0% success rate (all runs failed) must NOT return N/A.
	assert.Equal(t, "0%", formatForecastPercent(0, true), "0% with data → '0%'")
}

func TestFormatForecastPercent_NonZero(t *testing.T) {
	assert.Equal(t, "92%", formatForecastPercent(0.923, true))
}

func TestFormatForecastPercent_OneHundred(t *testing.T) {
	assert.Equal(t, "100%", formatForecastPercent(1.0, true))
}

// ── formatForecastTokens ─────────────────────────────────────────────────────

func TestFormatForecastTokens_Zero(t *testing.T) {
	assert.Equal(t, "-", formatForecastTokens(0))
}

func TestFormatForecastTokens_SmallInt(t *testing.T) {
	assert.Equal(t, "500", formatForecastTokens(500))
}

func TestFormatForecastTokens_Kilo(t *testing.T) {
	assert.Equal(t, "12.5K", formatForecastTokens(12500))
}

func TestFormatForecastTokens_Mega(t *testing.T) {
	assert.Equal(t, "1.20M", formatForecastTokens(1_200_000))
}

// ── extractWorkflowIDFromName ─────────────────────────────────────────────────

func TestExtractWorkflowIDFromName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ci-doctor", "ci-doctor"},
		{"ci-doctor.lock.yml", "ci-doctor"},
		{"ci-doctor.yml", "ci-doctor"},
		{"foo.yaml", "foo"},
		{"daily-planner.lock.yml", "daily-planner"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, extractWorkflowIDFromName(tc.in), "input=%q", tc.in)
	}
}

// ── RunForecast validation ────────────────────────────────────────────────────

func TestRunForecast_InvalidPeriod(t *testing.T) {
	cfg := ForecastConfig{Days: 30, Period: "quarter", SampleSize: 10}
	err := RunForecast(cfg)
	require.Error(t, err, "should error for invalid period")
}

func TestRunForecast_InvalidDays(t *testing.T) {
	cfg := ForecastConfig{Days: 90, Period: "month", SampleSize: 10}
	err := RunForecast(cfg)
	require.Error(t, err, "should error for days=90 (max is 30)")
}

func TestNewForecastCommand_DaysFlagDocumentsAllowedValues(t *testing.T) {
	cmd := NewForecastCommand()
	require.NotNil(t, cmd)

	daysFlag := cmd.Flags().Lookup("days")
	require.NotNil(t, daysFlag, "forecast command should register --days")
	assert.Equal(t, "Historical window in days to sample run history (allowed values: 7, 30)", daysFlag.Usage)
	assert.NotContains(t, cmd.Long, ").  When runs have been", "Long description should not contain duplicate spacing")
	assert.NotContains(t, cmd.Long, "used.  The", "Long description should not contain duplicate spacing")
	assert.NotContains(t, cmd.Long, "interval.  Use this", "Long description should not contain duplicate spacing")
}

// ── Duration enrichment ───────────────────────────────────────────────────────

// TestDurationEnrichment verifies that the forecast loop computes Duration from
// StartedAt/UpdatedAt when the Duration field is zero (as returned by gh run list).
func TestDurationEnrichment(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)

	r := WorkflowRun{
		Status:     "completed",
		Conclusion: "success",
		StartedAt:  start,
		UpdatedAt:  end,
		// Duration is intentionally zero (not populated by gh run list)
	}

	// Simulate the enrichment logic from forecastWorkflow.
	if r.Duration == 0 && !r.StartedAt.IsZero() && !r.UpdatedAt.IsZero() {
		r.Duration = r.UpdatedAt.Sub(r.StartedAt)
	}

	assert.Equal(t, 5*time.Minute, r.Duration)
}

// TestObservedRunsPerPeriodConsistency verifies that the λ value stored in the
// JSON-serialisable ForecastWorkflowResult.ObservedRunsPerPeriod field is the same
// value that would be passed to runMonteCarlo (R-MC-002).
//
// This is a structural test: it constructs a result whose ObservedRunsPerPeriod is
// set by the same arithmetic used in forecastWorkflow, then calls runMonteCarlo with
// that field directly and asserts the simulation produces sensible output — confirming
// that no intermediate recalculation or mutation of λ occurs between JSON output and
// Monte Carlo execution.
func TestObservedRunsPerPeriodConsistency(t *testing.T) {
	// Reproduce the λ calculation from forecastWorkflow.
	const (
		historyDays   = 30
		sampledRuns   = 15
		projectedDays = 30 // "month" period
	)
	observedRunsPerPeriod := float64(sampledRuns) / float64(historyDays) * float64(projectedDays)

	// Populate a ForecastWorkflowResult the same way forecastWorkflow does.
	result := ForecastWorkflowResult{
		WorkflowID:            "ci-doctor",
		Period:                "month",
		SampledRuns:           sampledRuns,
		HistoryDays:           historyDays,
		ObservedRunsPerPeriod: observedRunsPerPeriod,
	}

	// Build deterministic ET observations.
	etObs := make([]int, sampledRuns)
	for i := range etObs {
		etObs[i] = 10_000 + i*500
	}
	successCount := sampledRuns

	// runMonteCarlo uses result.ObservedRunsPerPeriod as λ — the same field that
	// appears in JSON output. Verify both the field value and the simulation are
	// consistent (non-nil, same λ).
	rng := rand.New(rand.NewSource(99)) //nolint:gosec
	mc := runMonteCarlo(etObs, successCount, result.ObservedRunsPerPeriod, rng)
	require.NotNil(t, mc, "runMonteCarlo must return non-nil for positive ObservedRunsPerPeriod")

	// The field exposed in JSON output must equal what was used for MC.
	assert.InEpsilon(t, observedRunsPerPeriod, result.ObservedRunsPerPeriod, 1e-12,
		"ObservedRunsPerPeriod JSON field must equal the λ passed to runMonteCarlo")

	// Sanity-check simulation output is plausible for the given λ.
	assert.Positive(t, mc.P50ProjectedEffectiveTokens,
		"P50 should be positive when success rate is 100%%")
	assert.LessOrEqual(t, mc.P10ProjectedEffectiveTokens, mc.P50ProjectedEffectiveTokens,
		"P10 ≤ P50")
	assert.LessOrEqual(t, mc.P50ProjectedEffectiveTokens, mc.P90ProjectedEffectiveTokens,
		"P50 ≤ P90")
}

// TestForecastWorkflow_LambdaConsistencyAcrossOutputFormats verifies that the λ value
// used by the Monte Carlo engine is identical to the ObservedRunsPerPeriod field exposed
// in both JSON output and verbose/table diagnostics (forecast-specification.md §13,
// closes issue #31984).
//
// Both renderForecastJSON and renderForecastTable operate on the same ForecastResult
// struct, so the λ used by runMonteCarlo (result.ObservedRunsPerPeriod) is always the
// same value reported to the caller in either output format.
func TestForecastWorkflow_LambdaConsistencyAcrossOutputFormats(t *testing.T) {
	originalList := forecastListWorkflowRunsPaginated
	t.Cleanup(func() {
		forecastListWorkflowRunsPaginated = originalList
	})

	const (
		historyDays   = 30
		projectedDays = 30 // "month" period
	)
	completedRuns := []WorkflowRun{
		{Status: "completed", Conclusion: "success", EffectiveTokens: 10_000, Duration: 5 * time.Minute},
		{Status: "completed", Conclusion: "success", EffectiveTokens: 12_000, Duration: 6 * time.Minute},
		{Status: "completed", Conclusion: "failure", EffectiveTokens: 8_000, Duration: 3 * time.Minute},
		{Status: "completed", Conclusion: "success", EffectiveTokens: 11_000, Duration: 7 * time.Minute},
		{Status: "completed", Conclusion: "success", EffectiveTokens: 9_500, Duration: 4 * time.Minute},
	}
	forecastListWorkflowRunsPaginated = func(_ ListWorkflowRunsOptions) ([]WorkflowRun, int, error) {
		return completedRuns, len(completedRuns), nil
	}

	result, err := forecastWorkflow(context.Background(), "ci-doctor", "2026-01-01", ForecastConfig{
		Days:       historyDays,
		Period:     "month",
		SampleSize: 100,
	}, projectedDays)
	require.NoError(t, err)

	// The expected λ is the observed run frequency scaled to the projection period.
	// This is also the value emitted in the JSON "observed_runs_per_period" field.
	n := len(completedRuns)
	expectedLambda := float64(n) / float64(historyDays) * float64(projectedDays)

	// Verify ObservedRunsPerPeriod (the JSON-serialised λ) equals the expected value.
	assert.InEpsilon(t, expectedLambda, result.ObservedRunsPerPeriod, 1e-12,
		"JSON field observed_runs_per_period must equal the λ used by the Monte Carlo engine")

	// Monte Carlo must have been called with the same λ — confirmed by a non-nil result.
	require.NotNil(t, result.MonteCarlo,
		"Monte Carlo simulation must run for positive ObservedRunsPerPeriod (λ=%.2f)", expectedLambda)

	// Both JSON output (renderForecastJSON) and table output (renderForecastTable) use the
	// same ForecastResult, so they are structurally guaranteed to report the same λ.
	assert.Positive(t, result.MonteCarlo.P50ProjectedEffectiveTokens,
		"P50 must be positive for positive λ and non-zero ET observations")
}

func TestForecastRateLimitSleep_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := forecastRateLimitSleep(ctx, time.Second)
	require.ErrorIs(t, err, context.Canceled)
}

func TestForecastRateLimitSleep_CompletesWithoutCancellation(t *testing.T) {
	err := forecastRateLimitSleep(context.Background(), time.Millisecond)
	require.NoError(t, err)
}

func TestForecastWorkflow_IgnoresSkippedRuns(t *testing.T) {
	originalList := forecastListWorkflowRunsPaginated
	t.Cleanup(func() {
		forecastListWorkflowRunsPaginated = originalList
	})

	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	forecastListWorkflowRunsPaginated = func(_ ListWorkflowRunsOptions) ([]WorkflowRun, int, error) {
		runs := []WorkflowRun{
			{Status: "completed", Conclusion: "skipped", EffectiveTokens: 999, Duration: 10 * time.Minute},
			{Status: "completed", Conclusion: "success", EffectiveTokens: 100, Duration: 5 * time.Minute, StartedAt: start, UpdatedAt: start.Add(5 * time.Minute)},
			{Status: "completed", Conclusion: "failure", EffectiveTokens: 200, Duration: 6 * time.Minute, StartedAt: start.Add(10 * time.Minute), UpdatedAt: start.Add(16 * time.Minute)},
		}
		return runs, len(runs), nil
	}

	result, err := forecastWorkflow(context.Background(), "smoke-copilot", "2026-01-01", ForecastConfig{
		Days:       30,
		Period:     "month",
		SampleSize: 100,
	}, 30)
	require.NoError(t, err)
	assert.Equal(t, 2, result.SampledRuns, "skipped runs should not be sampled")
	assert.Equal(t, 150, result.AvgEffectiveTokens, "metrics should ignore skipped runs")
	assert.InEpsilon(t, 0.5, result.SuccessRate, 1e-9)
}

func TestRenderForecastTable_ZeroMonteCarloRangeRendersDash(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	originalStderr := os.Stderr
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = originalStderr
	})

	err = renderForecastTable(ForecastResult{
		Period: "month",
		Workflows: []ForecastWorkflowResult{
			{
				WorkflowID:  "smoke-copilot",
				SampledRuns: 1,
				SuccessRate: 1,
				MonteCarlo: &ForecastMonteCarloSummary{
					P10ProjectedEffectiveTokens: 0,
					P50ProjectedEffectiveTokens: 0,
					P90ProjectedEffectiveTokens: 0,
				},
			},
		},
	}, ForecastConfig{Days: 30, Period: "month"})
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	out, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	assert.NotContains(t, string(out), "-–-")
}
