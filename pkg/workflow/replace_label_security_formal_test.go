//go:build !integration

// Package workflow – replace_label security & error-handling formal model tests.
//
// This file encodes the formal specification predicates for the security (§8)
// and error-handling (§7) surface of specs/replace-label-spec.md, which is not
// covered by replace_label_formal_test.go (schema/allowlist/staged/target
// resolution) or replace_label_transitions_formal_test.go (transitions and
// POST-setLabels verification).
//
// As with the sibling formal files, the helpers below are spec-level models of
// the semantics described in the specification, not wrappers around the
// production JavaScript handler (actions/setup/js/replace_label.cjs).
// Regressions in the handler itself are detected by the JavaScript test suite.
package workflow

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formalSanitizeLabelValue models RL-007: label values are trimmed and control
// characters are stripped before use.
func formalSanitizeLabelValue(value string) string {
	var b strings.Builder
	for _, r := range value {
		// C0 controls (including \t, \n, \r), DEL and C1 controls are removed.
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// formalRateLimitRetryEligible models RL-037/RL-048: which REST failures are
// eligible for the RATE_LIMIT_RETRY_CONFIG retry policy.
//   - HTTP 429 (primary rate limit) is always retryable.
//   - HTTP 403 (secondary rate limit) is retryable only when a Retry-After
//     header is present; a plain 403 is a permission error.
//   - All other statuses (e.g. 404, 422, 500) are not rate-limit retryable.
func formalRateLimitRetryEligible(status int, hasRetryAfter bool) bool {
	switch status {
	case 429:
		return true
	case 403:
		return hasRetryAfter
	default:
		return false
	}
}

type formalRetryOutcome struct {
	Success  bool
	Attempts int
	Error    string
}

// formalRunWithRateLimitRetry models RL-037: the retry loop is bounded by
// maxRetries, short-circuits on non-retryable errors, and hard-fails once the
// retry budget is exhausted.
//
// attempt returns (status, hasRetryAfter); status 200 denotes success.
func formalRunWithRateLimitRetry(maxRetries int, attempt func(n int) (int, bool)) formalRetryOutcome {
	for n := 1; n <= maxRetries+1; n++ {
		status, hasRetryAfter := attempt(n)
		if status == 200 {
			return formalRetryOutcome{Success: true, Attempts: n}
		}
		if !formalRateLimitRetryEligible(status, hasRetryAfter) {
			return formalRetryOutcome{
				Attempts: n,
				Error:    fmt.Sprintf("replace_label: setLabels failed with status %d", status),
			}
		}
	}
	return formalRetryOutcome{
		Attempts: maxRetries + 1,
		Error:    "replace_label: setLabels failed after exhausting rate-limit retries",
	}
}

// formalRESTLabelArray models RL-041/RL-042: the labels array sent to
// issues.setLabels is the current label set minus label_to_remove, plus
// label_to_add, deduplicated with label_to_add present exactly once.
func formalRESTLabelArray(current []string, labelToRemove, labelToAdd string) []string {
	return formalComputeNewLabelSet(current, labelToRemove, labelToAdd)
}

// formalSetLabelsResult models RL-046: a non-2xx setLabels response is a hard
// error yielding {success:false, error:<non-empty>}.
func formalSetLabelsResult(status int) formalReplaceLabelOutcome {
	if status >= 200 && status < 300 {
		return formalReplaceLabelOutcome{Success: true}
	}
	return formalReplaceLabelOutcome{Success: false}
}

func formalSetLabelsError(status int) string {
	if status >= 200 && status < 300 {
		return ""
	}
	return fmt.Sprintf("replace_label: setLabels REST call failed with HTTP %d", status)
}

// formalLabelMustPreExist models RL-052: the implementation never creates
// labels; a label_to_add absent from the repository is a hard error.
func formalLabelMustPreExist(repoLabels []string, labelToAdd string) (bool, string) {
	if slices.Contains(repoLabels, labelToAdd) {
		return true, ""
	}
	return false, fmt.Sprintf("replace_label: label_to_add %q does not exist in the target repository", labelToAdd)
}

// formalServerSideEnforcedDecision models RL-049: allow/blocklist evaluation is
// performed server-side and any bypass claim carried in the agent message is
// ignored.
func formalServerSideEnforcedDecision(labelToAdd string, allowed, blocked []string, agentClaimedBypass bool) bool {
	_ = agentClaimedBypass
	return formalValidateSingleLabel(labelToAdd, allowed, blocked, "label_to_add") == nil
}

// formalTokenScopeSatisfied models RL-054/RL-051: issues:write is the minimum
// required scope for replace-label (including cross-repository operation).
func formalTokenScopeSatisfied(scopes []string) bool {
	return slices.Contains(scopes, "issues:write")
}

func TestFormalSec_P1_SanitizedLabelValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "surrounding whitespace trimmed", input: "  bug  ", expected: "bug"},
		{name: "newline stripped", input: "bu\ng", expected: "bug"},
		{name: "carriage return and tab stripped", input: "\tbug\r\n", expected: "bug"},
		{name: "null byte stripped", input: "bu\x00g", expected: "bug"},
		{name: "ansi escape stripped", input: "\x1b[31mbug\x1b[0m", expected: "[31mbug[0m"},
		{name: "already clean value unchanged", input: "needs-triage", expected: "needs-triage"},
		{name: "internal spaces preserved", input: " good first issue ", expected: "good first issue"},
		{name: "control-only value collapses to empty", input: "\x00\x01\x02", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formalSanitizeLabelValue(tt.input))
		})
	}
}

func TestFormalSec_P1_SanitizationPrecedesRequiredCheck(t *testing.T) {
	// A value consisting solely of whitespace/control characters must fail the
	// RL-004/RL-005 non-empty check after sanitization.
	assert.False(t, formalRequiredNonEmptyLabel(formalSanitizeLabelValue("  \n\t ")))
	assert.True(t, formalRequiredNonEmptyLabel(formalSanitizeLabelValue(" bug\n")))
}

func TestFormalSec_P2_RateLimitRetryEligible(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		hasRetryAfter bool
		expected      bool
	}{
		{name: "429 always retries", status: 429, hasRetryAfter: false, expected: true},
		{name: "429 with retry-after retries", status: 429, hasRetryAfter: true, expected: true},
		{name: "403 with retry-after retries", status: 403, hasRetryAfter: true, expected: true},
		{name: "403 without retry-after is permission error", status: 403, hasRetryAfter: false, expected: false},
		{name: "422 never retries", status: 422, hasRetryAfter: true, expected: false},
		{name: "404 never retries", status: 404, hasRetryAfter: false, expected: false},
		{name: "500 is not rate-limit retryable", status: 500, hasRetryAfter: true, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formalRateLimitRetryEligible(tt.status, tt.hasRetryAfter))
		})
	}
}

func TestFormalSec_P3_RateLimitRetryBoundedRetries(t *testing.T) {
	t.Run("succeeds within budget", func(t *testing.T) {
		outcome := formalRunWithRateLimitRetry(5, func(n int) (int, bool) {
			if n < 3 {
				return 429, false
			}
			return 200, false
		})
		assert.True(t, outcome.Success)
		assert.Equal(t, 3, outcome.Attempts)
		assert.Empty(t, outcome.Error)
	})

	t.Run("hard fails when budget exhausted", func(t *testing.T) {
		outcome := formalRunWithRateLimitRetry(5, func(int) (int, bool) { return 429, false })
		assert.False(t, outcome.Success)
		assert.Equal(t, 6, outcome.Attempts)
		require.NotEmpty(t, outcome.Error)
	})

	t.Run("short-circuits on non-retryable error", func(t *testing.T) {
		outcome := formalRunWithRateLimitRetry(5, func(int) (int, bool) { return 422, false })
		assert.False(t, outcome.Success)
		assert.Equal(t, 1, outcome.Attempts, "non-retryable failures must not consume the retry budget")
		require.NotEmpty(t, outcome.Error)
	})
}

func TestFormalSec_P4_NewLabelSetContainsAddExactlyOnce(t *testing.T) {
	tests := []struct {
		name     string
		current  []string
		remove   string
		add      string
		expected []string
	}{
		{name: "basic replace", current: []string{"todo", "triaged"}, remove: "todo", add: "done", expected: []string{"triaged", "done"}},
		{name: "remove absent", current: []string{"triaged"}, remove: "todo", add: "done", expected: []string{"triaged", "done"}},
		{name: "add already present", current: []string{"todo", "done"}, remove: "todo", add: "done", expected: []string{"done"}},
		{name: "add equals remove", current: []string{"todo", "triaged"}, remove: "todo", add: "todo", expected: []string{"triaged", "todo"}},
		{name: "empty current set", current: nil, remove: "todo", add: "done", expected: []string{"done"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := formalRESTLabelArray(tt.current, tt.remove, tt.add)
			assert.Equal(t, tt.expected, labels)
			assert.Equal(t, 1, countOccurrences(labels, tt.add), "label_to_add must appear exactly once")
			if tt.remove != tt.add {
				assert.NotContains(t, labels, tt.remove, "label_to_remove must be excluded")
			}
			deduped := sortedCopy(labels)
			assert.Len(t, slices.Compact(deduped), len(labels), "labels array must be deduplicated")
		})
	}
}

func countOccurrences(labels []string, target string) int {
	count := 0
	for _, l := range labels {
		if l == target {
			count++
		}
	}
	return count
}

func sortedCopy(labels []string) []string {
	out := slices.Clone(labels)
	slices.Sort(out)
	return out
}

func TestFormalSec_P5_RESTFailureIsHardError(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 410, 422, 429, 500, 502, 503} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			outcome := formalSetLabelsResult(status)
			assert.False(t, outcome.Success)
			assert.False(t, outcome.Skipped, "REST failures are hard errors, not soft skips")
			require.NotEmpty(t, formalSetLabelsError(status))
		})
	}

	t.Run("status 200 succeeds", func(t *testing.T) {
		outcome := formalSetLabelsResult(200)
		assert.True(t, outcome.Success)
		assert.Empty(t, formalSetLabelsError(200))
	})
}

func TestFormalSec_P6_LabelMustPreExist(t *testing.T) {
	repoLabels := []string{"todo", "in-progress", "done"}

	t.Run("existing label accepted", func(t *testing.T) {
		ok, errMsg := formalLabelMustPreExist(repoLabels, "done")
		assert.True(t, ok)
		assert.Empty(t, errMsg)
	})

	t.Run("missing label is hard error", func(t *testing.T) {
		ok, errMsg := formalLabelMustPreExist(repoLabels, "brand-new")
		assert.False(t, ok)
		require.NotEmpty(t, errMsg)
		assert.Contains(t, errMsg, "brand-new")
		assert.Len(t, repoLabels, 3, "implementation must not create the missing label")
	})

	t.Run("empty repository label set rejects any add", func(t *testing.T) {
		ok, errMsg := formalLabelMustPreExist(nil, "done")
		assert.False(t, ok)
		require.NotEmpty(t, errMsg)
	})
}

func TestFormalSec_P7_ServerSideEnforcementOnly(t *testing.T) {
	allowed := []string{"status/*"}
	blocked := []string{"status/secret"}

	tests := []struct {
		name     string
		label    string
		expected bool
	}{
		{name: "allowed label", label: "status/done", expected: true},
		{name: "blocked label", label: "status/secret", expected: false},
		{name: "label outside allowlist", label: "priority/high", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withoutClaim := formalServerSideEnforcedDecision(tt.label, allowed, blocked, false)
			withClaim := formalServerSideEnforcedDecision(tt.label, allowed, blocked, true)
			assert.Equal(t, tt.expected, withoutClaim)
			assert.Equal(t, withoutClaim, withClaim, "agent-claimed bypass must have no effect on the decision")
		})
	}
}

func TestFormalSec_P8_TokenScopeMinimum(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		expected bool
	}{
		{name: "issues write present", scopes: []string{"issues:write"}, expected: true},
		{name: "issues write among others", scopes: []string{"contents:read", "issues:write"}, expected: true},
		{name: "issues read only", scopes: []string{"issues:read"}, expected: false},
		{name: "unrelated scopes", scopes: []string{"contents:write", "pull-requests:write"}, expected: false},
		{name: "no scopes", scopes: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formalTokenScopeSatisfied(tt.scopes))
		})
	}
}
