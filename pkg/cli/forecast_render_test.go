//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
