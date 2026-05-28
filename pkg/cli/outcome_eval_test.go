//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	s := ComputeOutcomeSummary(reports)

	assert.Equal(t, 6, s.Total, "total should count all reports")
	assert.Equal(t, 2, s.Accepted, "accepted count")
	assert.Equal(t, 1, s.Rejected, "rejected count")
	assert.Equal(t, 1, s.Ignored, "ignored count")
	assert.Equal(t, 1, s.Pending, "pending count")
	assert.Equal(t, 1, s.Lifecycle, "lifecycle count")
	assert.Equal(t, 1, s.ZeroTouch, "zero-touch count")
	assert.Equal(t, 0, s.AcceptedStrong, "accepted strong count")
	assert.Equal(t, 2, s.AcceptedMedium, "accepted medium count")
	assert.Equal(t, 0, s.AcceptedWeak, "accepted weak count")

	// AcceptanceRate = accepted / (accepted + rejected) = 2/3
	assert.InDelta(t, 0.6667, s.AcceptanceRate, 0.01, "acceptance rate")

	// WasteRate = rejected / total = 1/6
	assert.InDelta(t, 0.1667, s.WasteRate, 0.01, "waste rate")

	// ZeroTouchRate = zero_touch / accepted = 1/2
	assert.InDelta(t, 0.5, s.ZeroTouchRate, 0.01, "zero-touch rate")

	// MedianTimeToOutcome of [2.0, 8.0, 24.0] = 8.0
	assert.InDelta(t, 8.0, s.MedianTimeToOutcome, 0.01, "median time to outcome")
}

func TestComputeOutcomeSummaryEmpty(t *testing.T) {
	s := ComputeOutcomeSummary(nil)

	assert.Equal(t, 0, s.Total, "empty total")
	assert.InDelta(t, 0.0, s.AcceptanceRate, 1e-12, "empty acceptance rate")
	assert.InDelta(t, 0.0, s.WasteRate, 1e-12, "empty waste rate")
	assert.InDelta(t, 0.0, s.ZeroTouchRate, 1e-12, "empty zero-touch rate")
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
	assert.InDelta(t, 0.0, medianFloat(nil), 1e-12, "empty slice")
	assert.InDelta(t, 5.0, medianFloat([]float64{5.0}), 1e-12, "single element")
	assert.InDelta(t, 3.0, medianFloat([]float64{1.0, 3.0, 5.0}), 1e-12, "odd count")
	assert.InDelta(t, 2.5, medianFloat([]float64{1.0, 2.0, 3.0, 4.0}), 1e-12, "even count")
	assert.InDelta(t, 3.0, medianFloat([]float64{5.0, 1.0, 3.0}), 1e-12, "unsorted")
}

func TestTimeBetween(t *testing.T) {
	hours := timeBetween("2026-05-12T00:00:00Z", "2026-05-12T02:30:00Z")
	assert.InDelta(t, 2.5, hours, 0.01, "2.5 hours between timestamps")

	assert.InDelta(t, 0.0, timeBetween("bad", "2026-05-12T00:00:00Z"), 1e-12, "bad from timestamp")
	assert.InDelta(t, 0.0, timeBetween("2026-05-12T00:00:00Z", "bad"), 1e-12, "bad to timestamp")
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

func TestNormalizeOutcomeEvaluationTargetExistsOnly(t *testing.T) {
	report := OutcomeReport{
		Type:   "add_labels",
		Result: OutcomeUnknown,
		Detail: "object still exists",
	}

	eval := normalizeOutcomeEvaluation(report)
	assert.Equal(t, OutcomeStatusUnknown, eval.OutcomeStatus)
	assert.Equal(t, EvidenceWeak, eval.EvidenceStrength)
	assert.Equal(t, "target_exists_only", eval.Signal)
}

func TestEvalGenericStickyTargetExistsOnlyFallback(t *testing.T) {
	old := genericOutcomeGHAPIGet
	t.Cleanup(func() {
		genericOutcomeGHAPIGet = old
	})
	genericOutcomeGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"state": "open"}, nil
	}

	report := evalGenericSticky(
		CreatedItemReport{Type: "add_labels", Number: 42, Repo: "owner/repo"},
		"owner/repo",
	)

	assert.Equal(t, OutcomeUnknown, report.Result)
	assert.Equal(t, OutcomeStatusUnknown, report.OutcomeStatus)
	assert.Equal(t, EvidenceWeak, report.EvidenceStrength)
	assert.Equal(t, "target_exists_only", report.Signal)
}

func TestOutcomeSummaryExcludesExistsOnlyFromAccepted(t *testing.T) {
	reports := []OutcomeReport{
		{
			Type:   "add_labels",
			Result: OutcomeUnknown,
			OutcomeEvaluation: OutcomeEvaluation{
				OutcomeStatus:    OutcomeStatusUnknown,
				EvidenceStrength: EvidenceWeak,
				Signal:           "target_exists_only",
			},
		},
		{
			Type:   "create_pull_request",
			Result: OutcomeAccepted,
			OutcomeEvaluation: OutcomeEvaluation{
				OutcomeStatus:    OutcomeStatusAccepted,
				EvidenceStrength: EvidenceStrong,
				Signal:           "merged",
			},
		},
	}

	s := ComputeOutcomeSummary(reports)
	assert.Equal(t, 1, s.Accepted)
	assert.Equal(t, 1, s.AcceptedStrong)
	assert.Equal(t, 0, s.AcceptedWeak)
	assert.Equal(t, 1, s.FallbackExistsOnlyCount)
}

func TestWriteOutcomeJSONLEmitsNormalizedFields(t *testing.T) {
	dir := t.TempDir()
	reports := []OutcomeReport{
		{
			Type:   "add_labels",
			Result: OutcomeUnknown,
			OutcomeEvaluation: OutcomeEvaluation{
				OutcomeStatus:    OutcomeStatusUnknown,
				EvidenceStrength: EvidenceWeak,
				Signal:           "target_exists_only",
			},
			CreatedAt: "2026-05-12T00:00:00Z",
			CheckedAt: "2026-05-12T01:00:00Z",
		},
	}

	writeOutcomeJSONL(dir, 123, reports)

	data, err := os.ReadFile(filepath.Join(dir, "outcomes-123.jsonl"))
	require.NoError(t, err)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(data), &entry))
	assert.Equal(t, "unknown", entry["outcome_status"])
	assert.Equal(t, "weak", entry["evidence_strength"])
	assert.Equal(t, "target_exists_only", entry["signal"])
}

func TestEvalAddReviewerAcceptedWithApproval(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"users": []any{}, "teams": []any{}}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{
			{
				"state":        "APPROVED",
				"submitted_at": "2026-05-12T01:00:00Z",
				"user":         map[string]any{"login": "reviewer1"},
			},
		}, nil
	}

	report := evalAddReviewer(CreatedItemReport{
		Type:      "add_reviewer",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T00:00:00Z",
		Metadata: map[string]any{
			"requested_reviewers": []any{"reviewer1"},
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Equal(t, OutcomeStatusAccepted, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "review_approved", report.Signal)
}

func TestEvalAddReviewerRejectedWhenRequestRemoved(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"users": []any{}, "teams": []any{}}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{}, nil
	}

	report := evalAddReviewer(CreatedItemReport{
		Type:      "add_reviewer",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T00:00:00Z",
		Metadata: map[string]any{
			"requested_reviewers": []any{"reviewer1"},
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeRejected, report.Result)
	assert.Equal(t, OutcomeStatusRejected, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "review_request_removed", report.Signal)
}

func TestEvalSubmitPullRequestReviewDismissed(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"state": "open", "merged": false}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{
			{"id": float64(101), "state": "DISMISSED", "submitted_at": "2026-05-12T01:00:00Z"},
		}, nil
	}

	report := evalSubmitPullRequestReview(CreatedItemReport{
		Type:      "submit_pull_request_review",
		URL:       "https://github.com/owner/repo/pull/42#pullrequestreview-101",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T01:00:00Z",
		Metadata:  map[string]any{"review_id": float64(101)},
	}, "owner/repo")

	assert.Equal(t, OutcomeRejected, report.Result)
	assert.Equal(t, OutcomeStatusRejected, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "review_dismissed", report.Signal)
}

func TestEvalSubmitPullRequestReviewChangesRequestedMergedAfterPush(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"state":     "closed",
			"merged":    true,
			"merged_at": "2026-05-12T05:00:00Z",
		}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		switch endpoint {
		case "pulls/42/reviews":
			return []map[string]any{
				{"id": float64(101), "state": "CHANGES_REQUESTED", "submitted_at": "2026-05-12T02:00:00Z"},
			}, nil
		case "pulls/42/commits":
			return []map[string]any{
				{"commit": map[string]any{"committer": map[string]any{"date": "2026-05-12T03:00:00Z"}}},
			}, nil
		default:
			return []map[string]any{}, nil
		}
	}

	report := evalSubmitPullRequestReview(CreatedItemReport{
		Type:      "submit_pull_request_review",
		URL:       "https://github.com/owner/repo/pull/42#pullrequestreview-101",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T02:00:00Z",
		Metadata:  map[string]any{"review_id": float64(101)},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Equal(t, OutcomeStatusAccepted, report.OutcomeStatus)
	assert.Equal(t, EvidenceMedium, report.EvidenceStrength)
	assert.Equal(t, "changes_requested_addressed", report.Signal)
}

func TestEvalSubmitPullRequestReviewPendingWhenLatestOnOpenPR(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"state": "open", "merged": false}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{
			{"id": float64(100), "state": "COMMENTED", "submitted_at": "2026-05-12T00:30:00Z"},
			{"id": float64(101), "state": "COMMENTED", "submitted_at": "2026-05-12T01:00:00Z"},
		}, nil
	}

	report := evalSubmitPullRequestReview(CreatedItemReport{
		Type:      "submit_pull_request_review",
		URL:       "https://github.com/owner/repo/pull/42#pullrequestreview-101",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T01:00:00Z",
		Metadata:  map[string]any{"review_id": float64(101)},
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result)
	assert.Equal(t, OutcomeStatusPending, report.OutcomeStatus)
	assert.Equal(t, EvidenceMedium, report.EvidenceStrength)
	assert.Equal(t, "latest_review_pending", report.Signal)
}

func TestEvalAddReviewerPendingWhenRequestStillOutstanding(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"users": []any{map[string]any{"login": "reviewer1"}},
			"teams": []any{},
		}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{}, nil
	}

	report := evalAddReviewer(CreatedItemReport{
		Type:      "add_reviewer",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T00:00:00Z",
		Metadata: map[string]any{
			"requested_reviewers": []any{"reviewer1"},
		},
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result)
	assert.Equal(t, OutcomeStatusPending, report.OutcomeStatus)
	assert.Equal(t, EvidenceMedium, report.EvidenceStrength)
	assert.Equal(t, "awaiting_review", report.Signal)
}

func TestEvalAddReviewerUsesLatestReviewerState(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"users": []any{}, "teams": []any{}}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{
			{"state": "APPROVED", "submitted_at": "2026-05-12T01:00:00Z", "user": map[string]any{"login": "reviewer1"}},
			{"state": "CHANGES_REQUESTED", "submitted_at": "2026-05-12T02:00:00Z", "user": map[string]any{"login": "reviewer1"}},
		}, nil
	}

	report := evalAddReviewer(CreatedItemReport{
		Type:      "add_reviewer",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T00:00:00Z",
		Metadata: map[string]any{
			"requested_reviewers": []any{"reviewer1"},
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Equal(t, OutcomeStatusAccepted, report.OutcomeStatus)
	assert.Equal(t, EvidenceMedium, report.EvidenceStrength)
	assert.Equal(t, "review_submitted", report.Signal)
}

func TestTimestampOnOrAfterMalformedReturnsFalse(t *testing.T) {
	assert.False(t, timestampOnOrAfter("invalid", "2026-05-12T00:00:00Z"))
	assert.False(t, timestampOnOrAfter("2026-05-12T00:00:00Z", "invalid"))
}

func TestTimestampOnOrAfterEmptyCandidateAndThresholdHandling(t *testing.T) {
	assert.False(t, timestampOnOrAfter("", "2026-05-12T00:00:00Z"))
	assert.True(t, timestampOnOrAfter("2026-05-12T00:00:00Z", ""))
}

func TestEvalSubmitPullRequestReviewChangesRequestedMissingCommitDatesStaysUnknown(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"state":     "closed",
			"merged":    true,
			"merged_at": "2026-05-12T05:00:00Z",
		}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		switch endpoint {
		case "pulls/42/reviews":
			return []map[string]any{
				{"id": float64(101), "state": "CHANGES_REQUESTED", "submitted_at": "2026-05-12T02:00:00Z"},
			}, nil
		case "pulls/42/commits":
			return []map[string]any{
				{"commit": map[string]any{"committer": map[string]any{"date": ""}, "author": map[string]any{"date": ""}}},
			}, nil
		default:
			return []map[string]any{}, nil
		}
	}

	report := evalSubmitPullRequestReview(CreatedItemReport{
		Type:      "submit_pull_request_review",
		URL:       "https://github.com/owner/repo/pull/42#pullrequestreview-101",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T02:00:00Z",
		Metadata:  map[string]any{"review_id": float64(101)},
	}, "owner/repo")

	assert.Equal(t, OutcomeUnknown, report.Result)
	assert.Equal(t, OutcomeStatusUnknown, report.OutcomeStatus)
	assert.Equal(t, EvidenceWeak, report.EvidenceStrength)
	assert.Equal(t, "unknown", report.Signal)
}

func TestEvalSubmitPullRequestReviewApprovedMergedUsesSharedSignal(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"state":     "closed",
			"merged":    true,
			"merged_at": "2026-05-12T05:00:00Z",
		}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{
			{"id": float64(101), "state": "APPROVED", "submitted_at": "2026-05-12T02:00:00Z"},
		}, nil
	}

	report := evalSubmitPullRequestReview(CreatedItemReport{
		Type:      "submit_pull_request_review",
		URL:       "https://github.com/owner/repo/pull/42#pullrequestreview-101",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T02:00:00Z",
		Metadata:  map[string]any{"review_id": float64(101)},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Equal(t, OutcomeStatusAccepted, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "review_approved", report.Signal)
}

func TestEvalSubmitPullRequestReviewPendingIgnoresUnsubmittedDrafts(t *testing.T) {
	oldGet := outcomeReviewGHAPIGet
	oldGetArray := outcomeReviewGHAPIGetArray
	t.Cleanup(func() {
		outcomeReviewGHAPIGet = oldGet
		outcomeReviewGHAPIGetArray = oldGetArray
	})

	outcomeReviewGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{"state": "open", "merged": false}, nil
	}
	outcomeReviewGHAPIGetArray = func(endpoint string, repo string) ([]map[string]any, error) {
		return []map[string]any{
			{"id": float64(101), "state": "COMMENTED", "submitted_at": "2026-05-12T01:00:00Z"},
			{"id": float64(102), "state": "PENDING", "submitted_at": ""},
		}, nil
	}

	report := evalSubmitPullRequestReview(CreatedItemReport{
		Type:      "submit_pull_request_review",
		URL:       "https://github.com/owner/repo/pull/42#pullrequestreview-101",
		Number:    42,
		Repo:      "owner/repo",
		Timestamp: "2026-05-12T01:00:00Z",
		Metadata:  map[string]any{"review_id": float64(101)},
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result)
	assert.Equal(t, OutcomeStatusPending, report.OutcomeStatus)
	assert.Equal(t, EvidenceMedium, report.EvidenceStrength)
	assert.Equal(t, "latest_review_pending", report.Signal)
}
