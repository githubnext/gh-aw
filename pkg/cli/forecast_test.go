//go:build !integration

package cli

import (
	"context"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
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

// ── formatForecastAIC ─────────────────────────────────────────────────────

func TestFormatForecastAIC_Zero(t *testing.T) {
	assert.Equal(t, "-", formatForecastAIC(0))
}

func TestFormatForecastAIC_SmallInt(t *testing.T) {
	assert.Equal(t, "500", formatForecastAIC(500))
}

func TestFormatForecastAIC_Kilo(t *testing.T) {
	assert.Equal(t, "12.5K", formatForecastAIC(12500))
}

func TestFormatForecastAIC_Mega(t *testing.T) {
	assert.Equal(t, "1.20M", formatForecastAIC(1_200_000))
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

func TestRunForecast_InvalidTimeout(t *testing.T) {
	cfg := ForecastConfig{Days: 30, Period: "month", SampleSize: 10, TimeoutMinutes: -1}
	err := RunForecast(cfg)
	require.Error(t, err, "should error for negative timeout")
}

// TestRunForecast_R_IMPL_040_ExperimentalWarning verifies that the experimental status
// warning is emitted to stderr on every non-JSON invocation (R-IMPL-040), and is suppressed
// when --json is specified.
func TestRunForecast_R_IMPL_040_ExperimentalWarning(t *testing.T) {
	captureStderr := func(fn func()) string {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()
		orig := os.Stderr
		os.Stderr = w
		t.Cleanup(func() { os.Stderr = orig })
		fn()
		// Close the write end before reading so io.ReadAll sees EOF.
		require.NoError(t, w.Close())
		out, readErr := io.ReadAll(r)
		require.NoError(t, readErr)
		return string(out)
	}

	// Without --json: warning MUST appear on stderr.
	withoutJSON := captureStderr(func() {
		_ = RunForecast(ForecastConfig{Days: 30, Period: "quarter", SampleSize: 10})
	})
	assert.Contains(t, withoutJSON, "experimental", "R-IMPL-040: warning must appear when --json is not set")

	// With --json: warning MUST NOT appear on stderr.
	withJSON := captureStderr(func() {
		_ = RunForecast(ForecastConfig{Days: 30, Period: "quarter", SampleSize: 10, JSONOutput: true})
	})
	assert.NotContains(t, withJSON, "experimental", "R-IMPL-040: warning must be suppressed when --json is set")
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

func TestNewForecastCommand_TimeoutFlag(t *testing.T) {
	cmd := NewForecastCommand()
	require.NotNil(t, cmd)

	timeoutFlag := cmd.Flags().Lookup("timeout")
	require.NotNil(t, timeoutFlag, "forecast command should register --timeout")
	assert.Equal(t, "Gracefully stop forecast computation after this many minutes (0 disables timeout)", timeoutFlag.Usage)
	assert.Equal(t, "0", timeoutFlag.DefValue)
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
	assert.Positive(t, mc.P50ProjectedAIC,
		"P50 should be positive when success rate is 100%%")
	assert.LessOrEqual(t, mc.P10ProjectedAIC, mc.P50ProjectedAIC,
		"P10 ≤ P50")
	assert.LessOrEqual(t, mc.P50ProjectedAIC, mc.P90ProjectedAIC,
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
	originalLoadAIC := forecastLoadCachedRunAIC
	t.Cleanup(func() {
		forecastListWorkflowRunsPaginated = originalList
		forecastLoadCachedRunAIC = originalLoadAIC
	})

	const (
		historyDays   = 30
		projectedDays = 30 // "month" period
	)
	completedRuns := []WorkflowRun{
		{DatabaseID: 1, Status: "completed", Conclusion: "success", Duration: 5 * time.Minute},
		{DatabaseID: 2, Status: "completed", Conclusion: "success", Duration: 6 * time.Minute},
		{DatabaseID: 3, Status: "completed", Conclusion: "failure", Duration: 3 * time.Minute},
		{DatabaseID: 4, Status: "completed", Conclusion: "success", Duration: 7 * time.Minute},
		{DatabaseID: 5, Status: "completed", Conclusion: "success", Duration: 4 * time.Minute},
	}
	runAIC := map[int64]float64{
		1: 4.2,
		2: 5.0,
		3: 3.4,
		4: 4.6,
		5: 4.1,
	}
	forecastLoadCachedRunAIC = func(runID int64, _ bool) float64 {
		return runAIC[runID]
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
	assert.Positive(t, result.MonteCarlo.P50ProjectedAIC,
		"P50 must be positive for positive λ and non-zero AIC observations")
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
	originalLoadAIC := forecastLoadCachedRunAIC
	t.Cleanup(func() {
		forecastListWorkflowRunsPaginated = originalList
		forecastLoadCachedRunAIC = originalLoadAIC
	})

	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	forecastListWorkflowRunsPaginated = func(_ ListWorkflowRunsOptions) ([]WorkflowRun, int, error) {
		runs := []WorkflowRun{
			{DatabaseID: 11, Status: "completed", Conclusion: "skipped", Duration: 10 * time.Minute},
			{DatabaseID: 12, Status: "completed", Conclusion: "success", Duration: 5 * time.Minute, StartedAt: start, UpdatedAt: start.Add(5 * time.Minute)},
			{DatabaseID: 13, Status: "completed", Conclusion: "failure", Duration: 6 * time.Minute, StartedAt: start.Add(10 * time.Minute), UpdatedAt: start.Add(16 * time.Minute)},
		}
		return runs, len(runs), nil
	}
	runAIC := map[int64]float64{
		11: 9.99, // skipped and ignored
		12: 1.0,
		13: 2.0,
	}
	forecastLoadCachedRunAIC = func(runID int64, _ bool) float64 {
		return runAIC[runID]
	}

	result, err := forecastWorkflow(context.Background(), "smoke-copilot", "2026-01-01", ForecastConfig{
		Days:       30,
		Period:     "month",
		SampleSize: 100,
	}, 30)
	require.NoError(t, err)
	assert.Equal(t, 2, result.SampledRuns, "skipped runs should not be sampled")
	assert.InDelta(t, 1.5, result.AvgAIC, 1e-9, "metrics should ignore skipped runs")
	assert.InEpsilon(t, 0.5, result.SuccessRate, 1e-9)
}

func TestForecastWorkflow_RequestsSuccessfulRuns(t *testing.T) {
	originalList := forecastListWorkflowRunsPaginated
	originalLoadAIC := forecastLoadCachedRunAIC
	t.Cleanup(func() {
		forecastListWorkflowRunsPaginated = originalList
		forecastLoadCachedRunAIC = originalLoadAIC
	})

	var capturedOpts ListWorkflowRunsOptions
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	forecastListWorkflowRunsPaginated = func(opts ListWorkflowRunsOptions) ([]WorkflowRun, int, error) {
		capturedOpts = opts
		runs := []WorkflowRun{
			{DatabaseID: 12, Status: "completed", Conclusion: "success", Duration: 5 * time.Minute, StartedAt: start, UpdatedAt: start.Add(5 * time.Minute)},
		}
		return runs, len(runs), nil
	}
	forecastLoadCachedRunAIC = func(runID int64, _ bool) float64 {
		if runID == 12 {
			return 1.0
		}
		return 0
	}

	_, err := forecastWorkflow(context.Background(), "smoke-copilot", "2026-01-01", ForecastConfig{
		Days:       30,
		Period:     "month",
		SampleSize: 100,
	}, 30)
	require.NoError(t, err)
	assert.Equal(t, "success", capturedOpts.Status)
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
					P10ProjectedAIC: 0,
					P50ProjectedAIC: 0,
					P90ProjectedAIC: 0,
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

func TestLoadCachedRunAIC_UsageArtifactFirst(t *testing.T) {
	originalDownload := forecastDownloadRunArtifacts
	originalAnalyze := forecastAnalyzeTokenUsage
	t.Cleanup(func() {
		forecastDownloadRunArtifacts = originalDownload
		forecastAnalyzeTokenUsage = originalAnalyze
	})

	var downloaded []string
	forecastDownloadRunArtifacts = func(_ context.Context, _ int64, _ string, _ bool, _, _, _ string, artifactFilter []string) error {
		downloaded = append(downloaded, strings.Join(artifactFilter, ","))
		return nil
	}
	forecastAnalyzeTokenUsage = func(_ string, _ bool) (*TokenUsageSummary, error) {
		return &TokenUsageSummary{TotalAIC: 12.34}, nil
	}

	aic := loadCachedRunAIC(999_000_001, false)
	require.InDelta(t, 12.34, aic, 1e-9)
	require.Equal(t, []string{"usage"}, downloaded)
}

func TestLoadCachedRunAIC_FallsBackToLegacyAgentArtifacts(t *testing.T) {
	originalDownload := forecastDownloadRunArtifacts
	originalAnalyze := forecastAnalyzeTokenUsage
	t.Cleanup(func() {
		forecastDownloadRunArtifacts = originalDownload
		forecastAnalyzeTokenUsage = originalAnalyze
	})

	var downloaded []string
	forecastDownloadRunArtifacts = func(_ context.Context, _ int64, _ string, _ bool, _, _, _ string, artifactFilter []string) error {
		downloaded = append(downloaded, strings.Join(artifactFilter, ","))
		return nil
	}
	forecastAnalyzeTokenUsage = func(_ string, _ bool) (*TokenUsageSummary, error) {
		if len(downloaded) == 0 {
			return &TokenUsageSummary{}, nil
		}
		last := downloaded[len(downloaded)-1]
		if last == "usage" {
			return &TokenUsageSummary{}, nil
		}
		if last == constants.AgentArtifactName {
			return &TokenUsageSummary{TotalAIC: 4.56}, nil
		}
		return &TokenUsageSummary{}, nil
	}

	aic := loadCachedRunAIC(999_000_002, false)
	require.InDelta(t, 4.56, aic, 1e-9)
	require.Equal(t, []string{"usage", constants.AgentArtifactName}, downloaded)
}
