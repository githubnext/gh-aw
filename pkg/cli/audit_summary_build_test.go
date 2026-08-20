//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildProcessedAuditRun(t *testing.T) {
	run := WorkflowRun{DatabaseID: 11, WorkflowName: "Test Workflow"}
	results := auditAnalysisResults{
		firewallAnalysis:        &FirewallAnalysis{},
		policyAnalysis:          &PolicyAnalysis{},
		redactedDomainsAnalysis: &RedactedDomainsAnalysis{},
		missingTools:            []MissingToolReport{{Tool: "tool", Reason: "reason"}},
		missingData:             []MissingDataReport{{}},
		noops:                   []NoopReport{{}},
		mcpFailures:             []MCPFailureReport{{}},
		skillActivations:        []SkillActivation{{}},
		tokenUsageSummary:       &TokenUsageSummary{},
		rateLimitUsage:          &GitHubRateLimitUsage{},
		jobDetails:              []JobInfoWithDuration{{}},
	}

	processedRun := buildProcessedAuditRun(run, results)

	assert.Equal(t, run.DatabaseID, processedRun.Run.DatabaseID)
	assert.Same(t, results.firewallAnalysis, processedRun.FirewallAnalysis)
	assert.Same(t, results.policyAnalysis, processedRun.PolicyAnalysis)
	assert.Same(t, results.redactedDomainsAnalysis, processedRun.RedactedDomainsAnalysis)
	assert.Same(t, results.tokenUsageSummary, processedRun.TokenUsage)
	assert.Same(t, results.rateLimitUsage, processedRun.GitHubRateLimitUsage)
	assert.Len(t, processedRun.MissingTools, 1)
	assert.Len(t, processedRun.MissingData, 1)
	assert.Len(t, processedRun.Noops, 1)
	assert.Len(t, processedRun.MCPFailures, 1)
	assert.Len(t, processedRun.SkillActivations, 1)
	assert.Len(t, processedRun.JobDetails, 1)
}

func TestBuildAuditRunSummary(t *testing.T) {
	run := WorkflowRun{DatabaseID: 22}
	results := auditAnalysisResults{
		metrics:        LogMetrics{TokenUsage: 100, Turns: 3},
		accessAnalysis: &DomainAnalysis{},
		policyAnalysis: &PolicyAnalysis{},
		artifacts:      []string{"agent.log"},
	}
	taskDomain := &TaskDomainInfo{}
	processedRun := ProcessedRun{Run: run, TaskDomain: taskDomain}

	summary := buildAuditRunSummary(run, processedRun, results)

	require.NotNil(t, summary)
	assert.Equal(t, GetVersion(), summary.CLIVersion)
	assert.Equal(t, int64(22), summary.RunID)
	assert.False(t, summary.ProcessedAt.IsZero())
	assert.Equal(t, results.metrics, summary.Metrics)
	assert.Same(t, taskDomain, summary.TaskDomain)
	assert.Same(t, results.accessAnalysis, summary.AccessAnalysis)
	assert.Same(t, results.policyAnalysis, summary.PolicyAnalysis)
	assert.Equal(t, []string{"agent.log"}, summary.ArtifactsList)
}

func TestSaveAuditRunSummaryRoundTrip(t *testing.T) {
	outputDir := t.TempDir()
	run := WorkflowRun{DatabaseID: 33}
	results := auditAnalysisResults{metrics: LogMetrics{Turns: 4}}

	saveAuditRunSummary(outputDir, run, ProcessedRun{Run: run}, results, false)

	summary, ok := loadRunSummary(outputDir, false)
	require.True(t, ok)
	assert.Equal(t, int64(33), summary.RunID)
	assert.Equal(t, 4, summary.Metrics.Turns)
}
