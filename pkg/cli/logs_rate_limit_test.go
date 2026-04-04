//go:build !integration

package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRateLimitResponseUnmarshal verifies that the rateLimitResponse struct correctly
// unmarshals the JSON returned by `gh api rate_limit`.
func TestRateLimitResponseUnmarshal(t *testing.T) {
	now := time.Now().Add(30 * time.Second).Unix()
	raw := []byte(`{
		"resources": {
			"core": {
				"limit": 5000,
				"remaining": 42,
				"reset": ` + jsonInt(now) + `,
				"used": 4958
			}
		},
		"rate": {
			"limit": 5000,
			"remaining": 42,
			"reset": ` + jsonInt(now) + `,
			"used": 4958
		}
	}`)

	var resp rateLimitResponse
	require.NoError(t, json.Unmarshal(raw, &resp), "unmarshal should succeed")

	assert.Equal(t, 5000, resp.Resources.Core.Limit, "Limit should match")
	assert.Equal(t, 42, resp.Resources.Core.Remaining, "Remaining should match")
	assert.Equal(t, now, resp.Resources.Core.Reset, "Reset should match")
	assert.Equal(t, 4958, resp.Resources.Core.Used, "Used should match")
}

// TestRateLimitThresholdConstants verifies that the rate-limit constants are set to
// sensible values so a future edit that accidentally zeroes them will be caught.
func TestRateLimitThresholdConstants(t *testing.T) {
	assert.Positive(t, RateLimitThreshold, "RateLimitThreshold must be positive")
	assert.Positive(t, int64(APICallCooldown), "APICallCooldown must be positive")
	assert.Positive(t, int64(rateLimitResetBuffer), "rateLimitResetBuffer must be positive")
}

// TestRateLimitResourceIsBelowThreshold checks the boundary condition used by
// checkAndWaitForRateLimit: remaining <= RateLimitThreshold means we should wait.
func TestRateLimitResourceIsBelowThreshold(t *testing.T) {
	tests := []struct {
		name      string
		remaining int
		wantWait  bool
	}{
		{name: "zero remaining", remaining: 0, wantWait: true},
		{name: "exactly at threshold", remaining: RateLimitThreshold, wantWait: true},
		{name: "one above threshold", remaining: RateLimitThreshold + 1, wantWait: false},
		{name: "plenty remaining", remaining: 4000, wantWait: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := rateLimitResource{
				Limit:     5000,
				Remaining: tt.remaining,
				Reset:     time.Now().Add(60 * time.Second).Unix(),
				Used:      5000 - tt.remaining,
			}
			shouldWait := rl.Remaining <= RateLimitThreshold
			assert.Equal(t, tt.wantWait, shouldWait,
				"remaining=%d vs threshold=%d: wait mismatch", tt.remaining, RateLimitThreshold)
		})
	}
}

// jsonInt is a helper that converts an int64 to its JSON number representation.
func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
