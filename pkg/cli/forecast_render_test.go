//go:build !integration

package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForecastTableOmitsRetiredMetrics(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	originalStderr := os.Stderr
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = originalStderr
		reader.Close()
		writer.Close()
	})

	require.NoError(t, renderForecastTable(ForecastResult{
		Period: "month",
		Workflows: []ForecastWorkflowResult{{
			WorkflowID:          "daily-report",
			SampledRuns:         10,
			SuccessRate:         1,
			P50AIC:              1,
			P95AIC:              2,
			WeeklyProjectedAIC:  3,
			MonthlyProjectedAIC: 10,
		}},
	}, ForecastConfig{Days: 30, Period: "month"}))
	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	text := string(output)
	for _, header := range []string{"Workflow", "Runs", "P50 AIC/Run", "P95 AIC/Run", "Weekly AIC (P50)", "Monthly AIC (P50)", "Success Rate"} {
		assert.Contains(t, text, header)
	}
	assert.NotContains(t, strings.ToLower(text), "yield")
	assert.NotContains(t, strings.ToLower(text), "episode")
}

// ── formatForecastPercent ────────────────────────────────────────────────────

func TestFormatForecastPercent_NoData(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "N/A", formatForecastPercent(0, false), "no data → N/A")
}

func TestFormatForecastPercent_ZeroPercent(t *testing.T) {
	t.Parallel()
	// A legitimate 0% success rate (all runs failed) must NOT return N/A.
	assert.Equal(t, "0%", formatForecastPercent(0, true), "0% with data → '0%'")
}

func TestFormatForecastPercent_NonZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "92%", formatForecastPercent(0.923, true))
}

func TestFormatForecastPercent_OneHundred(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "100%", formatForecastPercent(1.0, true))
}

// ── formatForecastAIC ─────────────────────────────────────────────────────

func TestFormatForecastAIC_Zero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-", formatForecastAIC(0))
}

func TestFormatForecastAIC_SmallInt(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "500", formatForecastAIC(500))
}

func TestFormatForecastAIC_Kilo(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "12.5K", formatForecastAIC(12500))
}

func TestFormatForecastAIC_Mega(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.20M", formatForecastAIC(1_200_000))
}
