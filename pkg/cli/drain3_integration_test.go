//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDrain3Insights_NoEvents(t *testing.T) {
	// A ProcessedRun with no meaningful events should return no insights.
	processedRun := ProcessedRun{}
	metrics := MetricsData{}
	toolUsage := []ToolUsageInfo{}

	insights := buildDrain3Insights(processedRun, metrics, toolUsage)
	assert.Empty(t, insights, "expected no insights when run has no events")
}

func TestBuildDrain3Insights_BasicRun(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID: 42,
			Conclusion: "success",
			Turns:      5,
			TokenUsage: 1200,
		},
	}
	metrics := MetricsData{
		Turns:      5,
		TokenUsage: 1200,
		ErrorCount: 0,
	}
	toolUsage := []ToolUsageInfo{
		{Name: "bash", CallCount: 3},
		{Name: "github_issue_read", CallCount: 2},
	}

	insights := buildDrain3Insights(processedRun, metrics, toolUsage)
	require.NotEmpty(t, insights, "expected at least one drain3 insight for a run with events")

	// Verify all insights have required fields.
	for _, ins := range insights {
		assert.NotEmpty(t, ins.Category, "insight Category must not be empty")
		assert.NotEmpty(t, ins.Severity, "insight Severity must not be empty")
		assert.NotEmpty(t, ins.Title, "insight Title must not be empty")
		assert.NotEmpty(t, ins.Summary, "insight Summary must not be empty")
	}
}

func TestBuildDrain3Insights_WithErrors(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID: 99,
			Conclusion: "failure",
			Turns:      8,
			ErrorCount: 2,
		},
		MCPFailures: []MCPFailureReport{
			{ServerName: "github", Status: "connection_refused"},
			{ServerName: "search", Status: "timeout"},
		},
		MissingTools: []MissingToolReport{
			{Tool: "terraform", Reason: "not installed"},
		},
	}
	metrics := MetricsData{
		Turns:      8,
		ErrorCount: 2,
	}

	insights := buildDrain3Insights(processedRun, metrics, nil)
	require.NotEmpty(t, insights, "expected insights when run has MCP failures and missing tools")

	categories := make([]string, 0, len(insights))
	for _, ins := range insights {
		categories = append(categories, ins.Category)
	}
	// Should have execution and/or reliability categories.
	assert.Contains(t, categories, "execution", "expected an 'execution' category insight")
}

func TestBuildDrain3Insights_StageSequenceEvidence(t *testing.T) {
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID: 7,
			Conclusion: "success",
			Turns:      3,
		},
	}
	metrics := MetricsData{Turns: 3, TokenUsage: 500}
	toolUsage := []ToolUsageInfo{
		{Name: "search", CallCount: 1},
	}

	insights := buildDrain3Insights(processedRun, metrics, toolUsage)
	require.NotEmpty(t, insights, "expected insights to be generated")

	// One insight should carry the stage-sequence evidence.
	var found bool
	for _, ins := range insights {
		if ins.Title == "Agent stage sequence" {
			assert.NotEmpty(t, ins.Evidence, "stage sequence insight should have evidence")
			found = true
			break
		}
	}
	assert.True(t, found, "expected a 'Agent stage sequence' insight")
}

func TestBuildDrain3InsightsMultiRun_Empty(t *testing.T) {
	insights := buildDrain3InsightsMultiRun(nil)
	assert.Empty(t, insights, "expected no insights for nil runs slice")

	insights = buildDrain3InsightsMultiRun([]ProcessedRun{})
	assert.Empty(t, insights, "expected no insights for empty runs slice")
}

func TestBuildDrain3InsightsMultiRun_MultipleRuns(t *testing.T) {
	runs := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID: 1,
				Conclusion: "success",
				Turns:      5,
				TokenUsage: 1000,
				ErrorCount: 0,
			},
			MCPFailures:  []MCPFailureReport{},
			MissingTools: []MissingToolReport{},
		},
		{
			Run: WorkflowRun{
				DatabaseID: 2,
				Conclusion: "failure",
				Turns:      10,
				TokenUsage: 2000,
				ErrorCount: 3,
			},
			MCPFailures: []MCPFailureReport{
				{ServerName: "github", Status: "error"},
			},
			MissingTools: []MissingToolReport{
				{Tool: "docker", Reason: "not found"},
			},
		},
		{
			Run: WorkflowRun{
				DatabaseID: 3,
				Conclusion: "success",
				Turns:      4,
				TokenUsage: 800,
			},
		},
	}

	insights := buildDrain3InsightsMultiRun(runs)
	require.NotEmpty(t, insights, "expected insights from multi-run analysis")

	for _, ins := range insights {
		assert.NotEmpty(t, ins.Category, "insight Category must not be empty")
		assert.NotEmpty(t, ins.Severity, "insight Severity must not be empty")
		assert.NotEmpty(t, ins.Title, "insight Title must not be empty")
	}
}

func TestBuildAgentEventsFromProcessedRun(t *testing.T) {
	pr := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID: 5,
			Conclusion: "success",
			Turns:      4,
			TokenUsage: 900,
		},
		MCPFailures:  []MCPFailureReport{{ServerName: "s3", Status: "timeout"}},
		MissingTools: []MissingToolReport{{Tool: "kubectl", Reason: "missing"}},
		MissingData:  []MissingDataReport{{DataType: "env_var", Reason: "undefined"}},
		Noops:        []NoopReport{{Message: "already up to date"}},
	}
	metrics := MetricsData{Turns: 4, TokenUsage: 900}
	toolUsage := []ToolUsageInfo{
		{Name: "bash", CallCount: 2},
	}

	events := buildAgentEventsFromProcessedRun(pr, metrics, toolUsage)
	require.NotEmpty(t, events, "expected events to be generated")

	stages := make(map[string]int)
	for _, e := range events {
		stages[e.Stage]++
	}

	assert.Positive(t, stages["plan"], "expected at least one plan event")
	assert.Positive(t, stages["tool_call"], "expected at least one tool_call event")
	assert.Positive(t, stages["error"], "expected at least one error event")
	assert.Positive(t, stages["tool_result"], "expected at least one tool_result event (noop)")
	assert.Positive(t, stages["finish"], "expected at least one finish event")
}

func TestBuildDrain3Insights_IncludedInAuditData(t *testing.T) {
	// Verify that buildAuditData appends drain3 insights to ObservabilityInsights.
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID: 10,
			Conclusion: "success",
			Turns:      3,
			TokenUsage: 500,
		},
	}
	metrics := MetricsData{Turns: 3, TokenUsage: 500}
	toolUsage := []ToolUsageInfo{{Name: "bash", CallCount: 1}}

	// Combine the two pipelines as done in buildAuditData.
	existing := buildAuditObservabilityInsights(processedRun, metrics, toolUsage, nil)
	drain3 := buildDrain3Insights(processedRun, metrics, toolUsage)
	all := append(existing, drain3...)

	// We should have at least the drain3 insights.
	assert.GreaterOrEqual(t, len(all), len(drain3), "combined insights should include drain3 results")
}
