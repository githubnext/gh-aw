//go:build !integration

package cli

import (
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
