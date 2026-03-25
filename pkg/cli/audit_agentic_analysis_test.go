//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectTaskDomain(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			WorkflowName: "Weekly Research Report",
			WorkflowPath: ".github/workflows/weekly-research.yml",
			Event:        "schedule",
		},
	}

	domain := detectTaskDomain(processedRun, nil, nil, nil)
	require.NotNil(t, domain, "domain should be detected")
	assert.Equal(t, "research", domain.Name)
	assert.Equal(t, "Research", domain.Label)
}

func TestBuildAgenticAssessmentsFlagsPotentialDeterministicAlternative(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			WorkflowName: "Issue Triage",
			Turns:        2,
			Duration:     2 * time.Minute,
		},
	}
	metrics := MetricsData{Turns: 2}
	toolUsage := []ToolUsageInfo{{Name: "github_issue_read", CallCount: 1}}
	domain := &TaskDomainInfo{Name: "triage", Label: "Triage"}
	fingerprint := &BehaviorFingerprint{
		ExecutionStyle:  "directed",
		ToolBreadth:     "narrow",
		ActuationStyle:  "read_only",
		ResourceProfile: "lean",
		DispatchMode:    "standalone",
	}

	assessments := buildAgenticAssessments(processedRun, metrics, toolUsage, nil, domain, fingerprint, nil)
	require.NotEmpty(t, assessments)
	assert.Equal(t, "overkill_for_agentic", assessments[0].Kind)
}

func TestBuildAgenticAssessmentsFlagsResourceHeavyRun(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			WorkflowName:   "Deep Research",
			Turns:          15,
			Duration:       22 * time.Minute,
			SafeItemsCount: 4,
		},
	}
	metrics := MetricsData{Turns: 15}
	toolUsage := []ToolUsageInfo{
		{Name: "bash", CallCount: 4},
		{Name: "grep", CallCount: 3},
		{Name: "gh", CallCount: 2},
		{Name: "github_issue_read", CallCount: 2},
		{Name: "sed", CallCount: 1},
		{Name: "cat", CallCount: 1},
		{Name: "jq", CallCount: 1},
	}
	domain := &TaskDomainInfo{Name: "research", Label: "Research"}
	fingerprint := buildBehaviorFingerprint(processedRun, metrics, toolUsage, []CreatedItemReport{{Type: "create_issue"}}, nil)

	assessments := buildAgenticAssessments(processedRun, metrics, toolUsage, []CreatedItemReport{{Type: "create_issue"}}, domain, fingerprint, nil)

	var found bool
	for _, assessment := range assessments {
		if assessment.Kind == "resource_heavy_for_domain" {
			found = true
			assert.Equal(t, "high", assessment.Severity)
		}
	}
	assert.True(t, found, "resource heavy assessment should be present")
}

func TestBuildAuditDataIncludesAgenticAnalysis(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:   7,
			WorkflowName: "Issue Triage",
			WorkflowPath: ".github/workflows/issue-triage.yml",
			Status:       "completed",
			Conclusion:   "success",
			Duration:     3 * time.Minute,
			Turns:        3,
			Event:        "issues",
			LogsPath:     t.TempDir(),
		},
	}
	metrics := LogMetrics{Turns: 3}

	auditData := buildAuditData(processedRun, metrics, nil)
	require.NotNil(t, auditData.TaskDomain, "task domain should be present")
	require.NotNil(t, auditData.BehaviorFingerprint, "behavioral fingerprint should be present")
	assert.NotEmpty(t, auditData.AgenticAssessments, "agentic assessments should be present")
	assert.Equal(t, "triage", auditData.TaskDomain.Name)
}
