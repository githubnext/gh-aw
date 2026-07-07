// This file provides command-line interface functionality for gh-aw.
// This file (logs_rate_limit.go) contains helpers for querying the GitHub API
// rate limit and pausing execution when the remaining request budget is low.
//
// Key responsibilities:
//   - Fetching the current GitHub API rate limit via the gh CLI
//   - Sleeping until the rate-limit reset window when remaining requests are scarce
//   - Providing a drop-in replacement for the static APICallCooldown sleep used
//     between batch-fetch iterations in the logs orchestrator

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsRateLimitLog = logger.New("cli:logs_rate_limit")
var fetchRateLimitFunc = fetchRateLimit

func contextCause(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return context.Cause(ctx)
}

// rateLimitResponse models the JSON returned by `gh api rate_limit`.
// Only the "core" resource bucket is used because log downloads and
// workflow-run listing both draw from the core quota.
type rateLimitResponse struct {
	Resources struct {
		Core rateLimitResource `json:"core"`
	} `json:"resources"`
}

// rateLimitResource holds the fields relevant to a single GitHub API rate-limit bucket.
type rateLimitResource struct {
	// Limit is the maximum number of requests allowed per window.
	Limit int `json:"limit"`
	// Remaining is the number of requests still available in the current window.
	Remaining int `json:"remaining"`
	// Reset is the Unix timestamp (seconds) at which the window resets.
	Reset int64 `json:"reset"`
	// Used is the number of requests consumed so far in the current window.
	Used int `json:"used"`
}

// fetchRateLimit queries the GitHub API and returns the current core rate-limit
// state.  It is a thin wrapper around `gh api rate_limit` so that callers do
// not need to know about the CLI invocation details.
func fetchRateLimit() (rateLimitResource, error) {
	logsRateLimitLog.Print("Querying GitHub API rate limit")

	output, err := workflow.RunGHCombined("Verifying API quota...", "api", "rate_limit")
	if err != nil {
		return rateLimitResource{}, fmt.Errorf("failed to query rate limit: %w", err)
	}

	var resp rateLimitResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return rateLimitResource{}, fmt.Errorf("failed to parse rate limit response: %w", err)
	}

	logsRateLimitLog.Printf("Rate limit: limit=%d, remaining=%d, used=%d, reset=%d",
		resp.Resources.Core.Limit,
		resp.Resources.Core.Remaining,
		resp.Resources.Core.Used,
		resp.Resources.Core.Reset,
	)

	return resp.Resources.Core, nil
}

// sleepWithContext pauses for duration d and returns nil when the timer fires.
// If ctx is cancelled before the timer expires, it stops the timer and returns
// context.Cause(ctx) so callers can propagate cancellation (and any wrapped
// cause) immediately.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	var done <-chan struct{}
	if ctx != nil {
		done = ctx.Done()
	}
	// When ctx is nil, done remains nil. A nil channel is never selected, which
	// intentionally makes cancellation checks a no-op and preserves prior behavior.
	select {
	case <-done:
		return contextCause(ctx)
	default:
	}

	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-done:
		return contextCause(ctx)
	}
}

// checkAndWaitForRateLimit queries the GitHub API rate limit and sleeps until
// the reset window when the remaining core request budget falls at or below
// RateLimitThreshold.  It always waits at least APICallCooldown between
// successive calls so that even when requests are plentiful the orchestrator
// does not hammer the API.
//
// ctx is checked on every sleep so that a user cancellation (Ctrl-C) or
// deadline expiry wakes the function early and propagates the context error.
//
// If the rate limit cannot be fetched (e.g. network error) the function falls
// back to the static APICallCooldown sleep and returns the error so callers
// can decide whether to surface it.
func checkAndWaitForRateLimit(ctx context.Context, verbose bool) error {
	rl, err := fetchRateLimitFunc()
	if err != nil {
		// Best-effort: fall back to static cooldown so the caller can continue.
		logsRateLimitLog.Printf("Could not fetch rate limit, using static cooldown: %v", err)
		if sleepErr := sleepWithContext(ctx, APICallCooldown); sleepErr != nil {
			return fmt.Errorf("rate-limit fetch failed and context was canceled or timed out during fallback cooldown: %w", errors.Join(err, sleepErr))
		}
		return err
	}

	if rl.Remaining <= RateLimitThreshold {
		resetAt := time.Unix(rl.Reset, 0)
		waitDur := time.Until(resetAt)
		if waitDur <= 0 {
			// Reset has already passed; apply minimal cooldown and carry on.
			logsRateLimitLog.Print("Rate limit reset has already passed, applying minimal cooldown")
			return sleepWithContext(ctx, APICallCooldown)
		}

		// Add a small buffer so we don't resume right on the boundary.
		waitDur += rateLimitResetBuffer

		msg := fmt.Sprintf(
			"GitHub API rate limit nearly exhausted (%d of %d requests remaining). Waiting %.0f seconds until reset at %s",
			rl.Remaining, rl.Limit, waitDur.Seconds(), resetAt.UTC().Format(time.RFC3339),
		)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
		logsRateLimitLog.Printf("Sleeping for rate limit reset: duration=%s", waitDur)
		return sleepWithContext(ctx, waitDur)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(
			fmt.Sprintf("Rate limit OK: %d/%d requests remaining", rl.Remaining, rl.Limit),
		))
	}

	// Even when budget is healthy, apply the minimum inter-call cooldown.
	return sleepWithContext(ctx, APICallCooldown)
}
