//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvalUpdateIssueRetained(t *testing.T) {
	old := outcomeUpdateGHAPIGet
	t.Cleanup(func() {
		outcomeUpdateGHAPIGet = old
	})
	outcomeUpdateGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"title": "New title",
			"body":  "New body",
			"state": "open",
			"labels": []any{
				map[string]any{"name": "bug"},
				map[string]any{"name": "triage"},
			},
			"assignees": []any{
				map[string]any{"login": "octo"},
			},
		}, nil
	}

	report := evalUpdateIssue(CreatedItemReport{
		Type:   "update_issue",
		Number: 12,
		Repo:   "owner/repo",
		BeforeState: map[string]any{
			"title":     "Old title",
			"body_hash": mutableBodyHash("Old body"),
			"state":     "open",
			"labels":    []any{"triage"},
			"assignees": []any{},
		},
		AfterState: map[string]any{
			"title":     "New title",
			"body_hash": mutableBodyHash("New body"),
			"state":     "open",
			"labels":    []any{"triage", "bug"},
			"assignees": []any{"octo"},
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Equal(t, OutcomeStatusAccepted, report.OutcomeStatus)
	assert.Equal(t, EvidenceMedium, report.EvidenceStrength)
	assert.Equal(t, "state_retained", report.Signal)
}

func TestEvalUpdateIssueReverted(t *testing.T) {
	old := outcomeUpdateGHAPIGet
	t.Cleanup(func() {
		outcomeUpdateGHAPIGet = old
	})
	outcomeUpdateGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"title": "Old title",
			"body":  "Old body",
			"state": "open",
		}, nil
	}

	report := evalUpdateIssue(CreatedItemReport{
		Type:   "update_issue",
		Number: 12,
		Repo:   "owner/repo",
		BeforeState: map[string]any{
			"title":     "Old title",
			"body_hash": mutableBodyHash("Old body"),
			"state":     "open",
		},
		AfterState: map[string]any{
			"title":     "New title",
			"body_hash": mutableBodyHash("New body"),
			"state":     "closed",
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeRejected, report.Result)
	assert.Equal(t, OutcomeStatusRejected, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "state_reverted", report.Signal)
}

func TestEvalUpdatePullRequestRetainedAndMerged(t *testing.T) {
	old := outcomeUpdateGHAPIGet
	t.Cleanup(func() {
		outcomeUpdateGHAPIGet = old
	})
	outcomeUpdateGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"title":  "New title",
			"body":   "New body",
			"state":  "closed",
			"merged": true,
			"base":   map[string]any{"ref": "release"},
			"draft":  false,
			"head":   map[string]any{"sha": "def456"},
		}, nil
	}

	report := evalUpdatePullRequest(CreatedItemReport{
		Type:   "update_pull_request",
		Number: 42,
		Repo:   "owner/repo",
		BeforeState: map[string]any{
			"title":     "Old title",
			"body_hash": mutableBodyHash("Old body"),
			"state":     "open",
			"base":      "main",
			"draft":     true,
			"head_sha":  "abc123",
		},
		AfterState: map[string]any{
			"title":     "New title",
			"body_hash": mutableBodyHash("New body"),
			"state":     "closed",
			"base":      "release",
			"draft":     false,
			"head_sha":  "def456",
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Equal(t, OutcomeStatusAccepted, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "state_retained_and_merged", report.Signal)
}

func TestEvalUpdatePullRequestReplaced(t *testing.T) {
	old := outcomeUpdateGHAPIGet
	t.Cleanup(func() {
		outcomeUpdateGHAPIGet = old
	})
	outcomeUpdateGHAPIGet = func(endpoint string, repo string) (map[string]any, error) {
		return map[string]any{
			"title":  "Maintainer rewrite",
			"body":   "Reworked body",
			"state":  "open",
			"merged": false,
			"base":   map[string]any{"ref": "hotfix"},
			"draft":  false,
			"head":   map[string]any{"sha": "zzz999"},
		}, nil
	}

	report := evalUpdatePullRequest(CreatedItemReport{
		Type:   "update_pull_request",
		Number: 42,
		Repo:   "owner/repo",
		BeforeState: map[string]any{
			"title":     "Old title",
			"body_hash": mutableBodyHash("Old body"),
			"state":     "open",
			"base":      "main",
			"draft":     true,
			"head_sha":  "abc123",
		},
		AfterState: map[string]any{
			"title":     "New title",
			"body_hash": mutableBodyHash("New body"),
			"state":     "open",
			"base":      "release",
			"draft":     false,
			"head_sha":  "def456",
		},
	}, "owner/repo")

	assert.Equal(t, OutcomeRejected, report.Result)
	assert.Equal(t, OutcomeStatusRejected, report.OutcomeStatus)
	assert.Equal(t, EvidenceStrong, report.EvidenceStrength)
	assert.Equal(t, "state_replaced", report.Signal)
}

func TestEvalRetainedUpdateMissingExecutionStateUsesEvidenceNone(t *testing.T) {
	report := evalUpdateIssue(CreatedItemReport{
		Type:   "update_issue",
		Number: 12,
		Repo:   "owner/repo",
	}, "owner/repo")

	assert.Equal(t, OutcomeUnknown, report.Result)
	assert.Equal(t, OutcomeStatusUnknown, report.OutcomeStatus)
	assert.Equal(t, EvidenceNone, report.EvidenceStrength)
	assert.Equal(t, "missing_execution_state", report.Signal)
}
