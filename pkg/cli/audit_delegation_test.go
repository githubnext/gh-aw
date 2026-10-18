//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditDelegationFailureReporting(t *testing.T) {
	for _, tt := range []struct {
		name      string
		completed int
	}{
		{name: "failed delegation"},
		{name: "later completed delegation", completed: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			run := ProcessedRun{
				Run: WorkflowRun{Conclusion: "success", Turns: 8},
				TokenUsage: &TokenUsageSummary{
					SubagentModelRequests: []SubagentModelRequest{
						{AgentName: "reviewer", FailedCount: 1, CompletedCount: tt.completed},
						{AgentName: "researcher", FailedCount: 2},
						{AgentName: "successful", CompletedCount: 1},
					},
					DeclaredSubagentModels: []SubagentModelRequest{
						{AgentName: "reviewer", FailedCount: 1},
						{AgentName: "researcher", FailedCount: 2},
					},
				},
			}
			data, _ := buildLocalAuditData(run, LogMetrics{}, nil)
			assert.Equal(t, "success", data.Overview.Conclusion)
			assert.Zero(t, data.Metrics.ErrorCount, "do not change the workflow-error count")
			assert.Equal(t, 3, data.Metrics.SubagentFailureCount, "count invocations, not agents or mirrored declarations")

			failures := 0
			var success *AuditFinding
			for _, finding := range data.KeyFindings {
				switch finding.Code {
				case AuditFindingSubagentFailed:
					failures++
				case AuditFindingWorkflowSucceeded:
					success = &finding
				}
			}
			assert.Equal(t, 2, failures)
			require.NotNil(t, success)
			assert.Equal(t, "Parent Workflow Completed Successfully", success.Title)
			assert.Equal(t, "Parent workflow completed in 8 turns, but 3 sub-agent invocation(s) failed", success.Description)
			assert.Equal(t, "Parent success does not establish recovery or completion of failed delegated work; review sub-agent failures", success.Impact)
			assert.NotContains(t, success.Description, "no errors")
			assert.NotEqual(t, "No action needed", success.Impact)
			assertRecommendationExists(t, data.Recommendations, "high", "verify recovery")

			encoded, err := json.Marshal(data)
			require.NoError(t, err)
			var decoded AuditData
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			assert.Equal(t, 3, decoded.Metrics.SubagentFailureCount)
			_, stderr := captureOutput(t, func() error {
				renderConsoleMetrics(decoded.Metrics)
				return nil
			})
			assert.Contains(t, stderr, "errors=0")
			assert.Contains(t, stderr, "subagent_failures=3")
		})
	}
}

func TestAuditWithoutDelegationFailures(t *testing.T) {
	t.Parallel()
	for _, usage := range []*TokenUsageSummary{
		nil,
		{SubagentModelRequests: []SubagentModelRequest{{AgentName: "successful", CompletedCount: 1}}},
	} {
		data, _ := buildLocalAuditData(ProcessedRun{
			Run:        WorkflowRun{Conclusion: "success", Turns: 8},
			TokenUsage: usage,
		}, LogMetrics{}, nil)
		assert.Zero(t, data.Metrics.SubagentFailureCount)
		encoded, err := json.Marshal(data.Metrics)
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "subagent_failure_count")
		success := findFindingByCategory(data.KeyFindings, "success")
		require.NotNil(t, success)
		assert.Equal(t, AuditFindingWorkflowSucceeded, success.Code)
		assert.Equal(t, "Workflow Completed Successfully", success.Title)
		assert.Equal(t, "Completed in 8 turns with no reported workflow errors", success.Description)
		assert.Equal(t, "No action needed", success.Impact)
	}
}

func TestAuditDelegationFailuresPreserveWorkflowFailure(t *testing.T) {
	t.Parallel()
	data, _ := buildLocalAuditData(ProcessedRun{
		Run: WorkflowRun{Conclusion: "failure"},
		TokenUsage: &TokenUsageSummary{SubagentModelRequests: []SubagentModelRequest{
			{AgentName: "reviewer", FailedCount: 2},
		}},
	}, LogMetrics{}, nil)
	assert.Equal(t, "failure", data.Overview.Conclusion)
	assert.Equal(t, 1, data.Metrics.ErrorCount)
	assert.Equal(t, 2, data.Metrics.SubagentFailureCount)
	for _, finding := range data.KeyFindings {
		assert.NotEqual(t, AuditFindingWorkflowSucceeded, finding.Code)
	}
	assertFindingContains(t, data.KeyFindings, "error", "Workflow Failed")
	assertFindingContains(t, data.KeyFindings, "tooling", "Sub-agent Failed")
}

func TestAuditSuccessWithReportedErrorsIsQualified(t *testing.T) {
	t.Parallel()
	findings := generateSuccessFindings(WorkflowRun{Conclusion: "success"}, MetricsData{Turns: 8, ErrorCount: 1}, nil)
	require.Len(t, findings, 1)
	assert.Equal(t, "Workflow Completed Successfully", findings[0].Title)
	assert.Equal(t, "Workflow completed in 8 turns with 1 reported error(s)", findings[0].Description)
	assert.Equal(t, "Review reported errors despite the successful workflow conclusion", findings[0].Impact)
}

func TestAuditSuccessWithReportedErrorsAndDelegationFailuresIsQualified(t *testing.T) {
	t.Parallel()
	findings := generateSuccessFindings(WorkflowRun{Conclusion: "success"}, MetricsData{
		Turns:                8,
		ErrorCount:           2,
		SubagentFailureCount: 3,
	}, nil)
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingWorkflowSucceeded, findings[0].Code)
	assert.Equal(t, "Parent Workflow Completed Successfully", findings[0].Title)
	assert.Contains(t, findings[0].Description, "8 turns")
	assert.Contains(t, findings[0].Description, "2 reported error(s)")
	assert.Contains(t, findings[0].Description, "3 sub-agent invocation(s) failed")
	assert.Contains(t, findings[0].Impact, "Review reported errors")
	assert.Contains(t, findings[0].Impact, "does not establish recovery or completion of failed delegated work")
	assert.Contains(t, findings[0].Impact, "review sub-agent failures")
}
