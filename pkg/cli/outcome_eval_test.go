//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeOutcomeSummary(t *testing.T) {
	reports := []OutcomeReport{
		{Type: "create_pull_request", Result: OutcomeAccepted, ZeroTouch: true, TimeToOutcomeHours: 2.0},
		{Type: "create_pull_request", Result: OutcomeAccepted, ZeroTouch: false, TimeToOutcomeHours: 8.0},
		{Type: "create_issue", Result: OutcomeRejected, TimeToOutcomeHours: 24.0},
		{Type: "add_comment", Result: OutcomeIgnored},
		{Type: "assign_to_agent", Result: OutcomePending},
		{Type: "close_issue", Result: OutcomeLifecycle},
	}

	s := ComputeOutcomeSummary(reports, 10.0)

	assert.Equal(t, 6, s.Total, "total should count all reports")
	assert.Equal(t, 2, s.Accepted, "accepted count")
	assert.Equal(t, 1, s.Rejected, "rejected count")
	assert.Equal(t, 1, s.Ignored, "ignored count")
	assert.Equal(t, 1, s.Pending, "pending count")
	assert.Equal(t, 1, s.Lifecycle, "lifecycle count")
	assert.Equal(t, 1, s.ZeroTouch, "zero-touch count")

	// AcceptanceRate = accepted / (accepted + rejected) = 2/3
	assert.InDelta(t, 0.6667, s.AcceptanceRate, 0.01, "acceptance rate")

	// WasteRate = rejected / total = 1/6
	assert.InDelta(t, 0.1667, s.WasteRate, 0.01, "waste rate")

	// ZeroTouchRate = zero_touch / accepted = 1/2
	assert.InDelta(t, 0.5, s.ZeroTouchRate, 0.01, "zero-touch rate")

	// CostPerAcceptedOutcome = 10.0 / 2 = 5.0
	assert.InDelta(t, 5.0, s.CostPerAcceptedOutcome, 0.01, "cost per accepted outcome")

	// MedianTimeToOutcome of [2.0, 8.0, 24.0] = 8.0
	assert.InDelta(t, 8.0, s.MedianTimeToOutcome, 0.01, "median time to outcome")
}

func TestComputeOutcomeSummaryEmpty(t *testing.T) {
	s := ComputeOutcomeSummary(nil, 0)

	assert.Equal(t, 0, s.Total, "empty total")
	assert.Equal(t, 0.0, s.AcceptanceRate, "empty acceptance rate")
	assert.Equal(t, 0.0, s.WasteRate, "empty waste rate")
	assert.Equal(t, 0.0, s.ZeroTouchRate, "empty zero-touch rate")
}

func TestParseNumberFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected int
	}{
		{"PR URL", "https://github.com/owner/repo/pull/42", 42},
		{"issue URL", "https://github.com/owner/repo/issues/108", 108},
		{"comment URL", "https://github.com/owner/repo/issues/123#issuecomment-456", 123},
		{"empty", "", 0},
		{"no number", "https://github.com/owner/repo", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseNumberFromURL(tt.url), "parsed number from URL")
		})
	}
}

func TestParseRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"full URL", "https://github.com/owner/repo/pull/42", "owner/repo"},
		{"issues URL", "https://github.com/github/gh-aw/issues/123", "github/gh-aw"},
		{"no github", "https://example.com/foo", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseRepoFromURL(tt.url), "parsed repo from URL")
		})
	}
}

func TestNormalizeRepoForAPI(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		wantOwnerRepo string
		wantHost      string
	}{
		{"plain owner/repo", "owner/repo", "owner/repo", ""},
		{"GHES HOST/owner/repo", "myhost.com/owner/repo", "owner/repo", "myhost.com"},
		{"github.com/owner/repo treated as host prefix", "github.com/owner/repo", "owner/repo", "github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ownerRepo, host := normalizeRepoForAPI(tt.repo)
			assert.Equal(t, tt.wantOwnerRepo, ownerRepo, "owner/repo portion")
			assert.Equal(t, tt.wantHost, host, "host portion")
		})
	}
}

func TestIsBotUser(t *testing.T) {
	assert.True(t, isBotUser("github-actions[bot]"), "github-actions[bot] is a bot")
	assert.True(t, isBotUser("github-actions"), "github-actions is a bot")
	assert.True(t, isBotUser("copilot-swe-agent"), "copilot-swe-agent is a bot")
	assert.False(t, isBotUser("mnkiefer"), "human user is not a bot")
}

func TestExtractCommentID(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"issuecomment", "https://github.com/owner/repo/issues/123#issuecomment-456789", "456789"},
		{"comments path", "https://github.com/owner/repo/issues/comments/789012", "789012"},
		{"no comment", "https://github.com/owner/repo/issues/123", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractCommentID(tt.url), "extracted comment ID")
		})
	}
}

func TestResolveItemRepo(t *testing.T) {
	item := CreatedItemReport{Repo: "explicit/repo"}
	assert.Equal(t, "explicit/repo", resolveItemRepo(item, "fallback/repo"), "prefers item repo")

	item2 := CreatedItemReport{URL: "https://github.com/url/repo/pull/1"}
	assert.Equal(t, "url/repo", resolveItemRepo(item2, "fallback/repo"), "falls back to URL repo")

	item3 := CreatedItemReport{}
	assert.Equal(t, "fallback/repo", resolveItemRepo(item3, "fallback/repo"), "falls back to override")
}

func TestResolveItemNumber(t *testing.T) {
	item := CreatedItemReport{Number: 42}
	assert.Equal(t, 42, resolveItemNumber(item), "prefers item number")

	item2 := CreatedItemReport{URL: "https://github.com/owner/repo/pull/99"}
	assert.Equal(t, 99, resolveItemNumber(item2), "falls back to URL number")

	item3 := CreatedItemReport{}
	assert.Equal(t, 0, resolveItemNumber(item3), "returns 0 when no number")
}

func TestMedianFloat(t *testing.T) {
	assert.Equal(t, 0.0, medianFloat(nil), "empty slice")
	assert.Equal(t, 5.0, medianFloat([]float64{5.0}), "single element")
	assert.Equal(t, 3.0, medianFloat([]float64{1.0, 3.0, 5.0}), "odd count")
	assert.Equal(t, 2.5, medianFloat([]float64{1.0, 2.0, 3.0, 4.0}), "even count")
	assert.Equal(t, 3.0, medianFloat([]float64{5.0, 1.0, 3.0}), "unsorted")
}

func TestTimeBetween(t *testing.T) {
	hours := timeBetween("2026-05-12T00:00:00Z", "2026-05-12T02:30:00Z")
	assert.InDelta(t, 2.5, hours, 0.01, "2.5 hours between timestamps")

	assert.Equal(t, 0.0, timeBetween("bad", "2026-05-12T00:00:00Z"), "bad from timestamp")
	assert.Equal(t, 0.0, timeBetween("2026-05-12T00:00:00Z", "bad"), "bad to timestamp")
}

func TestEvaluateOutcomesSkipsNoopAndMetadata(t *testing.T) {
	items := []CreatedItemReport{
		{Type: "noop", Timestamp: "2026-05-12T00:00:00Z"},
		{Type: "missing_tool", Timestamp: "2026-05-12T00:00:00Z"},
		{Type: "missing_data", Timestamp: "2026-05-12T00:00:00Z"},
		{Type: "report_incomplete", Timestamp: "2026-05-12T00:00:00Z"},
	}

	reports := EvaluateOutcomes(items, "owner/repo")
	assert.Empty(t, reports, "noop and metadata types should be skipped")
}

func TestEvaluateOutcomesErrorOnMissingData(t *testing.T) {
	items := []CreatedItemReport{
		{Type: "create_pull_request", Timestamp: "2026-05-12T00:00:00Z"},
	}

	reports := EvaluateOutcomes(items, "")
	assert.Len(t, reports, 1, "should produce one report")
	assert.Equal(t, OutcomeError, reports[0].Result, "should error on missing repo and number")
}
