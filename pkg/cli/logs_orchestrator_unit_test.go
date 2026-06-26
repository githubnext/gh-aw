//go:build !integration

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsDeadlineExceeded verifies that the helper correctly identifies
// context.DeadlineExceeded and returns false for other cases (including nil error).
func TestIsDeadlineExceeded(t *testing.T) {
	t.Run("deadline exceeded context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond) // ensure deadline has fired
		assert.True(t, isDeadlineExceeded(ctx), "expected true for DeadlineExceeded context")
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.False(t, isDeadlineExceeded(ctx), "expected false for cancelled (not deadline) context")
	})

	t.Run("active context", func(t *testing.T) {
		ctx := context.Background()
		assert.False(t, isDeadlineExceeded(ctx), "expected false for active (non-cancelled) context")
	})
}

// TestNoRunsMessage verifies that the helper returns an informative message
// depending on the start_date filter and timeoutReached flag.
func TestNoRunsMessage(t *testing.T) {
	now := time.Now()
	futureDate := now.AddDate(0, 0, 5).Format("2006-01-02")
	oldDate := now.AddDate(0, 0, -100).Format("2006-01-02")
	recentDate := now.AddDate(0, 0, -5).Format("2006-01-02")
	futureRFC3339 := now.AddDate(1, 0, 0).Format(time.RFC3339)

	tests := []struct {
		name           string
		startDate      string
		timeoutReached bool
		wantContains   string
	}{
		{
			name:           "timeout reached",
			startDate:      "",
			timeoutReached: true,
			wantContains:   "Timeout reached",
		},
		{
			name:           "future date (YYYY-MM-DD)",
			startDate:      futureDate,
			timeoutReached: false,
			wantContains:   "is in the future",
		},
		{
			name:           "future date (RFC3339)",
			startDate:      futureRFC3339,
			timeoutReached: false,
			wantContains:   "is in the future",
		},
		{
			name:           "old date beyond retention",
			startDate:      oldDate,
			timeoutReached: false,
			wantContains:   "retention period",
		},
		{
			name:           "recent date within retention",
			startDate:      recentDate,
			timeoutReached: false,
			wantContains:   "No runs found matching",
		},
		{
			name:           "no start date",
			startDate:      "",
			timeoutReached: false,
			wantContains:   "No runs found matching",
		},
		{
			name:           "timeout takes priority over future date",
			startDate:      futureDate,
			timeoutReached: true,
			wantContains:   "Timeout reached",
		},
		{
			name:           "future date message includes the date value",
			startDate:      "2030-01-01",
			timeoutReached: false,
			wantContains:   "2030-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noRunsMessage(tt.startDate, tt.timeoutReached)
			assert.Contains(t, got, tt.wantContains,
				"noRunsMessage(%q, %v) = %q, want to contain %q", tt.startDate, tt.timeoutReached, got, tt.wantContains)
		})
	}
}

// TestParseFilterDate verifies that date strings accepted by the logs flags are
// correctly parsed into time.Time values.
func TestParseFilterDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"YYYY-MM-DD", "2024-01-15", false},
		{"RFC3339", "2024-01-15T10:30:00Z", false},
		{"RFC3339 with offset", "2024-01-15T10:30:00+05:00", false},
		{"invalid", "not-a-date", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFilterDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.False(t, got.IsZero(), "expected non-zero time")
			}
		})
	}
}
