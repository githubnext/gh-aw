//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAuditComparison_NoBaseline(t *testing.T) {
	comparison := buildAuditComparison(auditComparisonSnapshot{Turns: 4, Posture: "read_only"}, nil, nil)
	require.NotNil(t, comparison, "comparison should still be returned when no baseline exists")
	assert.False(t, comparison.BaselineFound, "baseline should be marked unavailable")
	assert.Nil(t, comparison.Baseline, "baseline details should be omitted")
	assert.Nil(t, comparison.Delta, "delta should be omitted when no baseline exists")
	assert.Nil(t, comparison.Classification, "classification should be omitted when no baseline exists")
}

func TestBuildAuditComparison_RiskyChange(t *testing.T) {
	baselineRun := &WorkflowRun{
		DatabaseID:   100,
		WorkflowName: "triage",
		Conclusion:   "success",
		CreatedAt:    time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
	}

	comparison := buildAuditComparison(
		auditComparisonSnapshot{Turns: 11, Posture: "write_capable", BlockedRequests: 7, MCPFailures: []string{"github"}},
		baselineRun,
		&auditComparisonSnapshot{Turns: 4, Posture: "read_only", BlockedRequests: 0},
	)

	require.NotNil(t, comparison, "comparison should be built")
	require.True(t, comparison.BaselineFound, "baseline should be marked available")
	require.NotNil(t, comparison.Delta, "delta should be present")
	require.NotNil(t, comparison.Classification, "classification should be present")
	require.NotNil(t, comparison.Recommendation, "recommendation should be present")

	assert.Equal(t, "risky", comparison.Classification.Label, "write-capable transition should be risky")
	assert.Contains(t, comparison.Classification.ReasonCodes, "turns_increase")
	assert.Contains(t, comparison.Classification.ReasonCodes, "posture_changed")
	assert.Contains(t, comparison.Classification.ReasonCodes, "blocked_requests_increase")
	assert.Contains(t, comparison.Classification.ReasonCodes, "new_mcp_failure")
	assert.Equal(t, 4, comparison.Delta.Turns.Before)
	assert.Equal(t, 11, comparison.Delta.Turns.After)
	assert.Equal(t, "read_only", comparison.Delta.Posture.Before)
	assert.Equal(t, "write_capable", comparison.Delta.Posture.After)
	assert.True(t, comparison.Delta.MCPFailure.NewlyPresent, "new MCP failure should be marked")
	assert.Contains(t, comparison.Recommendation.Action, "write-capable", "recommendation should address the risky posture change")
}

func TestBuildAuditComparison_StableRun(t *testing.T) {
	baselineRun := &WorkflowRun{DatabaseID: 99, WorkflowName: "triage", Conclusion: "success", CreatedAt: time.Now().Add(-time.Hour)}
	comparison := buildAuditComparison(
		auditComparisonSnapshot{Turns: 4, Posture: "read_only", BlockedRequests: 0},
		baselineRun,
		&auditComparisonSnapshot{Turns: 4, Posture: "read_only", BlockedRequests: 0},
	)

	require.NotNil(t, comparison.Classification, "classification should be present")
	assert.Equal(t, "stable", comparison.Classification.Label, "unchanged runs should be stable")
	assert.Empty(t, comparison.Classification.ReasonCodes, "stable runs should have no reason codes")
	assert.Contains(t, comparison.Recommendation.Action, "No action needed", "stable runs should produce a no-op recommendation")
}
