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
			assert.Contains(t, success.Description, "8 turns, but 3 sub-agent invocation(s) failed")
			assert.Contains(t, success.Impact, "does not establish recovery")
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
		assert.Equal(t, "Completed in 8 turns with no reported workflow errors", success.Description)
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
	assert.Contains(t, findings[0].Description, "1 reported error(s)")
	assert.NotEqual(t, "No action needed", findings[0].Impact)
}
