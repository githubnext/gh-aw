//go:build !integration

package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvalDispatchWorkflowNoRunID(t *testing.T) {
	// No metadata at all → pending with an informative detail message.
	report := evalDispatchWorkflow(CreatedItemReport{
		Type: "dispatch_workflow",
		Repo: "owner/repo",
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result)
	assert.Contains(t, report.Detail, "no run ID available")
}

func TestEvalDispatchWorkflowRunIDZero(t *testing.T) {
	// run_id present but zero → treated as missing.
	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": float64(0)},
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result)
}

func TestEvalDispatchWorkflowAPIError(t *testing.T) {
	old := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
	workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
		return nil, errors.New("connection refused")
	}

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": float64(12345678)},
	}, "owner/repo")

	assert.Equal(t, OutcomeError, report.Result)
	assert.Contains(t, report.EvalError, "connection refused")
}

func TestEvalDispatchWorkflowCompletedSuccess(t *testing.T) {
	old := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
	workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
		return map[string]any{
			"status":     "completed",
			"conclusion": "success",
		}, nil
	}

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": float64(12345678)},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
	assert.Contains(t, report.Detail, "success")
}

func TestEvalDispatchWorkflowCompletedFailure(t *testing.T) {
	for _, conclusion := range []string{"failure", "timed_out", "cancelled"} {
		t.Run(conclusion, func(t *testing.T) {
			old := workflowOutcomeGHAPIGet
			t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
			workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
				return map[string]any{
					"status":     "completed",
					"conclusion": conclusion,
				}, nil
			}

			report := evalDispatchWorkflow(CreatedItemReport{
				Type:     "dispatch_workflow",
				Repo:     "owner/repo",
				Metadata: map[string]any{"run_id": float64(99999)},
			}, "owner/repo")

			assert.Equal(t, OutcomeRejected, report.Result)
			assert.Contains(t, report.Detail, conclusion)
		})
	}
}

func TestEvalDispatchWorkflowCompletedOtherConclusion(t *testing.T) {
	old := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
	workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
		return map[string]any{
			"status":     "completed",
			"conclusion": "skipped",
		}, nil
	}

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": float64(42)},
	}, "owner/repo")

	assert.Equal(t, OutcomeIgnored, report.Result)
	assert.Contains(t, report.Detail, "skipped")
}

func TestEvalDispatchWorkflowInProgress(t *testing.T) {
	old := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
	workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
		return map[string]any{
			"status":     "in_progress",
			"conclusion": "",
		}, nil
	}

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": float64(77)},
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result)
	assert.Contains(t, report.Detail, "in_progress")
}

func TestEvalDispatchWorkflowRunIDInt64(t *testing.T) {
	// run_id supplied as int64 (not float64) must be handled without panic.
	old := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
	workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
		return map[string]any{
			"status":     "completed",
			"conclusion": "success",
		}, nil
	}

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": int64(9876543210)},
	}, "owner/repo")

	assert.Equal(t, OutcomeAccepted, report.Result)
}

func TestEvalDispatchWorkflowFloat64OverflowGuard(t *testing.T) {
	// A float64 value above 2^53 cannot represent integers exactly and must be
	// treated as an invalid run_id (OutcomePending) rather than silently truncated.
	// Use 2^53 + 2 (= 9007199254740994): consecutive integers around 2^53 collapse
	// to the same float64, so this value would be mangled if cast to int64 directly.
	aboveMaxSafeInt := float64(maxSafeFloat64Int) + 2

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": aboveMaxSafeInt},
	}, "owner/repo")

	assert.Equal(t, OutcomePending, report.Result,
		"a float64 run_id above 2^53 must be treated as invalid (OutcomePending)")
}

func TestEvalDispatchWorkflowActionRequired(t *testing.T) {
	// action_required is a blocking conclusion that requires manual intervention;
	// it must map to OutcomeRejected (not OutcomeIgnored).
	old := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = old })
	workflowOutcomeGHAPIGet = func(_ string, _ string) (map[string]any, error) {
		return map[string]any{
			"status":     "completed",
			"conclusion": "action_required",
		}, nil
	}

	report := evalDispatchWorkflow(CreatedItemReport{
		Type:     "dispatch_workflow",
		Repo:     "owner/repo",
		Metadata: map[string]any{"run_id": float64(12345678)},
	}, "owner/repo")

	assert.Equal(t, OutcomeRejected, report.Result,
		"action_required conclusion must map to OutcomeRejected")
	assert.Contains(t, report.Detail, "action_required")
}

func TestEvalUpdateDiscussionReturnsIgnored(t *testing.T) {
	// evalUpdateDiscussion must return OutcomeIgnored (not OutcomePending) so that
	// callers do not enter an infinite retry loop waiting for a terminal status.
	report := evalUpdateDiscussion(CreatedItemReport{
		Type: "update_discussion",
		URL:  "https://github.com/owner/repo/discussions/1",
	}, "owner/repo")

	assert.Equal(t, OutcomeIgnored, report.Result,
		"evalUpdateDiscussion must return OutcomeIgnored to prevent infinite retry")
	assert.NotEmpty(t, report.Detail)
}
