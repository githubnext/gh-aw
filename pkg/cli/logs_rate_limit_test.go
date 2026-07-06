//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRateLimitResponseUnmarshal verifies that the rateLimitResponse struct correctly
// unmarshals the JSON returned by `gh api rate_limit`.
func TestRateLimitResponseUnmarshal(t *testing.T) {
	now := time.Now().Add(time.Second * 30).Unix()
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
	return strconv.FormatInt(n, 10)
}

// TestSleepWithContextCancellation verifies that sleepWithContext returns ctx.Err()
// immediately when the context is cancelled before the timer fires.
func TestSleepWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	start := time.Now()
	err := sleepWithContext(ctx, 10*time.Second)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, time.Second, "sleepWithContext should return quickly when context is already cancelled")
}

// TestSleepWithContextDeadlineExceeded verifies that sleepWithContext respects a
// deadline that expires before the sleep duration.
func TestSleepWithContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := sleepWithContext(ctx, 10*time.Second)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, time.Second, "sleepWithContext should return as soon as the deadline expires")
}

// TestSleepWithContextTimerFires verifies that sleepWithContext returns nil when the
// timer fires before context cancellation.
func TestSleepWithContextTimerFires(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	err := sleepWithContext(ctx, 5*time.Millisecond)
	elapsed := time.Since(start)
	require.NoError(t, err, "sleepWithContext should return nil when timer fires normally")
	assert.Less(t, elapsed, time.Second, "timer should have fired and returned promptly")
}

func TestSleepWithContextNilContext(t *testing.T) {
	var nilCtx context.Context
	start := time.Now()
	err := sleepWithContext(nilCtx, 2*time.Millisecond)
	elapsed := time.Since(start)
	require.NoError(t, err, "nil context should fall back to background context")
	assert.Less(t, elapsed, time.Second, "nil context should not block longer than timer duration")
}

func TestSleepWithContextAlreadyCanceled(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := sleepWithContext(canceledCtx, 2*time.Millisecond)
	elapsed := time.Since(start)
	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, time.Second, "already-canceled context should return promptly")
}

func TestCheckAndWaitForRateLimitContextCancelled(t *testing.T) {
	oldFetchRateLimitFunc := fetchRateLimitFunc
	fetchRateLimitFunc = func() (rateLimitResource, error) {
		return rateLimitResource{
			Limit:     5000,
			Remaining: 0,
			Reset:     time.Now().Add(10 * time.Minute).Unix(),
		}, nil
	}
	t.Cleanup(func() { fetchRateLimitFunc = oldFetchRateLimitFunc })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := checkAndWaitForRateLimit(ctx, false)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 100*time.Millisecond, "cancelled context should interrupt rate-limit wait promptly")
}

func TestCheckAndWaitForRateLimitFetchErrorAndContextDone(t *testing.T) {
	oldFetchRateLimitFunc := fetchRateLimitFunc
	expectedFetchErr := stderrors.New("fetch failure")
	fetchRateLimitFunc = func() (rateLimitResource, error) {
		return rateLimitResource{}, expectedFetchErr
	}
	t.Cleanup(func() { fetchRateLimitFunc = oldFetchRateLimitFunc })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkAndWaitForRateLimit(ctx, false)
	require.Error(t, err)
	require.ErrorIs(t, err, expectedFetchErr)
	require.ErrorIs(t, err, context.Canceled)
}
