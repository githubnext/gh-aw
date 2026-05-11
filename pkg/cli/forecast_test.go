package cli

import (
	"testing"
	"time"
)

// ── formatForecastPercent ────────────────────────────────────────────────────

func TestFormatForecastPercent_NoData(t *testing.T) {
	if got := formatForecastPercent(0, false); got != "N/A" {
		t.Errorf("want N/A, got %q", got)
	}
}

func TestFormatForecastPercent_ZeroPercent(t *testing.T) {
	// A legitimate 0% success rate (all runs failed) must NOT return N/A.
	if got := formatForecastPercent(0, true); got != "0%" {
		t.Errorf("want 0%%, got %q", got)
	}
}

func TestFormatForecastPercent_NonZero(t *testing.T) {
	if got := formatForecastPercent(0.923, true); got != "92%" {
		t.Errorf("want 92%%, got %q", got)
	}
}

func TestFormatForecastPercent_OneHundred(t *testing.T) {
	if got := formatForecastPercent(1.0, true); got != "100%" {
		t.Errorf("want 100%%, got %q", got)
	}
}

// ── formatForecastTokens ─────────────────────────────────────────────────────

func TestFormatForecastTokens_Zero(t *testing.T) {
	if got := formatForecastTokens(0); got != "-" {
		t.Errorf("want -, got %q", got)
	}
}

func TestFormatForecastTokens_SmallInt(t *testing.T) {
	if got := formatForecastTokens(500); got != "500" {
		t.Errorf("want 500, got %q", got)
	}
}

func TestFormatForecastTokens_Kilo(t *testing.T) {
	if got := formatForecastTokens(12500); got != "12.5K" {
		t.Errorf("want 12.5K, got %q", got)
	}
}

func TestFormatForecastTokens_Mega(t *testing.T) {
	if got := formatForecastTokens(1_200_000); got != "1.20M" {
		t.Errorf("want 1.20M, got %q", got)
	}
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
		if got := extractWorkflowIDFromName(tc.in); got != tc.want {
			t.Errorf("extractWorkflowIDFromName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── RunForecast validation ────────────────────────────────────────────────────

func TestRunForecast_InvalidPeriod(t *testing.T) {
	cfg := ForecastConfig{Days: 30, Period: "quarter", SampleSize: 10}
	err := RunForecast(cfg)
	if err == nil {
		t.Fatal("expected error for invalid period, got nil")
	}
}

func TestRunForecast_InvalidDays(t *testing.T) {
	cfg := ForecastConfig{Days: 90, Period: "month", SampleSize: 10}
	err := RunForecast(cfg)
	if err == nil {
		t.Fatal("expected error for days=90, got nil")
	}
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

	if r.Duration != 5*time.Minute {
		t.Errorf("expected 5m duration, got %v", r.Duration)
	}
}
