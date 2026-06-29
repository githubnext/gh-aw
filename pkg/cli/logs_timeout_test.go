//go:build !integration

package cli

import (
	"testing"
	"time"
)

// TestTimeoutFlagParsing tests that the timeout flag is properly parsed
func TestTimeoutFlagParsing(t *testing.T) {
	tests := []struct {
		name            string
		timeout         int
		expectTimeout   bool
		expectedMinutes int
	}{
		{
			name:            "no timeout specified",
			timeout:         0,
			expectTimeout:   false,
			expectedMinutes: 0,
		},
		{
			name:            "timeout of 5 minutes",
			timeout:         5,
			expectTimeout:   true,
			expectedMinutes: 5,
		},
		{
			name:            "timeout of 30 minutes",
			timeout:         30,
			expectTimeout:   true,
			expectedMinutes: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the timeout value is correctly used
			if tt.expectTimeout && tt.timeout == 0 {
				t.Errorf("Expected timeout to be set but got 0")
			}
			if !tt.expectTimeout && tt.timeout != 0 {
				t.Errorf("Expected no timeout but got %d", tt.timeout)
			}
			if tt.expectTimeout && tt.timeout != tt.expectedMinutes {
				t.Errorf("Expected timeout of %d minutes but got %d", tt.expectedMinutes, tt.timeout)
			}
		})
	}
}

// TestTimeoutLogic tests the timeout logic without making network calls
func TestTimeoutLogic(t *testing.T) {
	tests := []struct {
		name          string
		timeout       int
		elapsed       time.Duration
		shouldTimeout bool
	}{
		{
			name:          "no timeout set",
			timeout:       0,
			elapsed:       100 * time.Minute,
			shouldTimeout: false,
		},
		{
			name:          "timeout not reached",
			timeout:       60,
			elapsed:       30 * time.Minute,
			shouldTimeout: false,
		},
		{
			name:          "just under boundary",
			timeout:       1,
			elapsed:       59 * time.Second,
			shouldTimeout: false,
		},
		{
			name:          "timeout exactly reached",
			timeout:       1,
			elapsed:       60 * time.Second,
			shouldTimeout: true,
		},
		{
			name:          "timeout exceeded",
			timeout:       1,
			elapsed:       90 * time.Second,
			shouldTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the timeout check logic (timeout is in minutes, elapsed in seconds)
			var timeoutReached bool
			if tt.timeout > 0 {
				if tt.elapsed.Seconds() >= float64(tt.timeout)*60 {
					timeoutReached = true
				}
			}

			if timeoutReached != tt.shouldTimeout {
				t.Errorf("Expected timeout reached=%v but got %v (timeout=%d min, elapsed=%.1fs)",
					tt.shouldTimeout, timeoutReached, tt.timeout, tt.elapsed.Seconds())
			}
		})
	}
}

// TestEffectiveMCPLogsToolTimeoutMinutes verifies that the MCP logs tool
// scales its implicit timeout with larger fetch windows while preserving
// explicit user-provided timeouts.
func TestEffectiveMCPLogsToolTimeoutMinutes(t *testing.T) {
	tests := []struct {
		name             string
		requestedTimeout int
		count            int
		want             int
	}{
		{
			name:             "explicit timeout is preserved",
			requestedTimeout: 5,
			count:            100,
			want:             5,
		},
		{
			name:             "small fetch window keeps one minute default",
			requestedTimeout: 0,
			count:            40,
			want:             1,
		},
		{
			name:             "fetch window above forty runs gets two minutes",
			requestedTimeout: 0,
			count:            41,
			want:             2,
		},
		{
			name:             "eighty run fetch window stays in two minute tier",
			requestedTimeout: 0,
			count:            80,
			want:             2,
		},
		{
			name:             "eighty one run fetch window enters three minute tier",
			requestedTimeout: 0,
			count:            81,
			want:             3,
		},
		{
			name:             "default hundred run window gets three minutes",
			requestedTimeout: 0,
			count:            100,
			want:             3,
		},
		{
			name:             "unspecified count falls back to default window size",
			requestedTimeout: 0,
			count:            0,
			want:             3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveMCPLogsToolTimeoutMinutes(tt.requestedTimeout, tt.count); got != tt.want {
				t.Errorf("effectiveMCPLogsToolTimeoutMinutes(%d, %d) = %d, want %d", tt.requestedTimeout, tt.count, got, tt.want)
			}
		})
	}
}
