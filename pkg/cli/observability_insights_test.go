//go:build !integration

package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAuditObservabilityInsights(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			Turns:          11,
			SafeItemsCount: 2,
		},
		MissingTools: []MissingToolReport{{Tool: "terraform"}},
		MCPFailures:  []MCPFailureReport{{ServerName: "github"}},
		MissingData:  []MissingDataReport{{DataType: "issue_body"}},
		FirewallAnalysis: &FirewallAnalysis{
			TotalRequests:   20,
			BlockedRequests: 8,
			AllowedRequests: 12,
		},
		RedactedDomainsAnalysis: &RedactedDomainsAnalysis{TotalDomains: 3},
	}

	metrics := MetricsData{Turns: 11}
	toolUsage := []ToolUsageInfo{
		{Name: "bash", CallCount: 4},
		{Name: "github_issue_read", CallCount: 2},
		{Name: "grep", CallCount: 1},
		{Name: "sed", CallCount: 1},
	}
	createdItems := []CreatedItemReport{{Type: "create_issue"}}

	insights := buildAuditObservabilityInsights(processedRun, metrics, toolUsage, createdItems)
	require.Len(t, insights, 5, "expected five audit insights from the supplied signals")

	titles := make([]string, 0, len(insights))
	for _, insight := range insights {
		titles = append(titles, insight.Title)
	}

	assert.Contains(t, titles, "Adaptive execution path")
	assert.Contains(t, titles, "Write path executed")
	assert.Contains(t, titles, "Capability friction detected")
	assert.Contains(t, titles, "Network friction detected")
	assert.Contains(t, titles, "Sensitive destinations were redacted")
}

func TestBuildLogsObservabilityInsights(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run:              WorkflowRun{WorkflowName: "triage", Conclusion: "failure", Turns: 3, SafeItemsCount: 0},
			MissingTools:     []MissingToolReport{{Tool: "terraform"}},
			FirewallAnalysis: &FirewallAnalysis{TotalRequests: 10, BlockedRequests: 1},
		},
		{
			Run:              WorkflowRun{WorkflowName: "triage", Conclusion: "failure", Turns: 9, SafeItemsCount: 1},
			MCPFailures:      []MCPFailureReport{{ServerName: "github"}},
			FirewallAnalysis: &FirewallAnalysis{TotalRequests: 10, BlockedRequests: 7},
		},
		{
			Run: WorkflowRun{WorkflowName: "docs", Conclusion: "success", Turns: 2, SafeItemsCount: 1},
		},
	}

	toolUsage := []ToolUsageSummary{
		{Name: "bash", TotalCalls: 14},
		{Name: "github_issue_read", TotalCalls: 6},
	}

	insights := buildLogsObservabilityInsights(processedRuns, toolUsage)
	require.NotEmpty(t, insights, "expected aggregated logs insights")

	var combined []string
	for _, insight := range insights {
		combined = append(combined, insight.Title+" "+insight.Summary)
	}
	text := strings.Join(combined, "\n")

	assert.Contains(t, text, "Failure hotspot identified")
	assert.Contains(t, text, "Execution drift observed")
	assert.Contains(t, text, "Capability hotspot identified")
	assert.Contains(t, text, "Network friction hotspot identified")
	assert.Contains(t, text, "Actuation mix summarized")
	assert.Contains(t, text, "Tool concentration observed")
}

func TestBuildAuditDataIncludesObservabilityInsights(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:     42,
			WorkflowName:   "insight-test",
			Status:         "completed",
			Conclusion:     "success",
			Duration:       2 * time.Minute,
			Turns:          7,
			SafeItemsCount: 1,
		},
	}

	metrics := workflow.LogMetrics{
		Turns: 7,
		ToolCalls: []workflow.ToolCallInfo{
			{Name: "bash", CallCount: 3},
			{Name: "github_issue_read", CallCount: 2},
			{Name: "grep", CallCount: 1},
			{Name: "sed", CallCount: 1},
		},
	}

	auditData := buildAuditData(processedRun, metrics, nil)
	require.NotEmpty(t, auditData.ObservabilityInsights, "audit data should expose observability insights")
	assert.Equal(t, "execution", auditData.ObservabilityInsights[0].Category)
}
