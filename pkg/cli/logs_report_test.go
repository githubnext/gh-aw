package cli

import (
	"testing"
	"time"
)

// TestRenderLogsConsoleUnified tests the unified console rendering
func TestRenderLogsConsoleUnified(t *testing.T) {
	// Create test data
	data := LogsData{
		Summary: LogsSummary{
			TotalRuns:         2,
			TotalDuration:     "10m30s",
			TotalTokens:       2500,
			TotalCost:         0.025,
			TotalTurns:        8,
			TotalErrors:       1,
			TotalWarnings:     3,
			TotalMissingTools: 2,
		},
		Runs: []RunData{
			{
				DatabaseID:       12345,
				WorkflowName:     "test-workflow",
				Status:           "completed",
				Duration:         "5m30s",
				TokenUsage:       1000,
				EstimatedCost:    0.01,
				Turns:            3,
				ErrorCount:       0,
				WarningCount:     2,
				MissingToolCount: 1,
				CreatedAt:        time.Now(),
				LogsPath:         "/tmp/logs/12345",
			},
		},
		ToolUsage: []ToolUsageSummary{
			{
				Name:             "github-mcp-server",
				TotalCalls:       1500,
				Runs:             5,
				MaxOutputSize:    2500000,
				MaxOutputDisplay: "2.4 MB",
				MaxDuration:      "1m30s",
			},
			{
				Name:             "playwright",
				TotalCalls:       500,
				Runs:             3,
				MaxOutputSize:    512000,
				MaxOutputDisplay: "500.0 KB",
				MaxDuration:      "45s",
			},
		},
		MissingTools: []MissingToolSummary{
			{
				Tool:               "terraform",
				Count:              5,
				Workflows:          []string{"workflow-a", "workflow-b", "workflow-c"},
				WorkflowsDisplay:   "workflow-a, workflow-b, workflow-c",
				FirstReason:        "Infrastructure automation needed",
				FirstReasonDisplay: "Infrastructure automation needed",
			},
			{
				Tool:               "kubectl",
				Count:              3,
				Workflows:          []string{"k8s-deploy"},
				WorkflowsDisplay:   "k8s-deploy",
				FirstReason:        "K8s management required",
				FirstReasonDisplay: "K8s management required",
			},
		},
		MCPFailures: []MCPFailureSummary{
			{
				ServerName:       "github-mcp-server",
				Count:            2,
				Workflows:        []string{"workflow-a", "workflow-b"},
				WorkflowsDisplay: "workflow-a, workflow-b",
			},
			{
				ServerName:       "playwright",
				Count:            1,
				Workflows:        []string{"browser-test"},
				WorkflowsDisplay: "browser-test",
			},
		},
		LogsLocation: "/tmp/logs",
	}

	// Test unified rendering - should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderLogsConsole panicked: %v", r)
		}
	}()

	renderLogsConsole(data, false)
	renderLogsConsole(data, true)
}

// TestBuildToolUsageSummaryPopulatesDisplay tests that display fields are populated
func TestBuildToolUsageSummaryPopulatesDisplay(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				LogsPath: "/tmp/test-logs",
			},
		},
	}

	result := buildToolUsageSummary(processedRuns)

	// If there are results, check that display fields are populated
	for _, tool := range result {
		if tool.MaxOutputSize > 0 && tool.MaxOutputDisplay == "" {
			t.Errorf("MaxOutputDisplay not populated for tool %s with MaxOutputSize %d", tool.Name, tool.MaxOutputSize)
		}
	}
}

// TestBuildMissingToolsSummaryPopulatesDisplay tests that display fields are populated
func TestBuildMissingToolsSummaryPopulatesDisplay(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				WorkflowName: "test-workflow",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "terraform",
					Reason:       "Infrastructure automation needed",
					WorkflowName: "test-workflow",
					RunID:        12345,
				},
			},
		},
	}

	result := buildMissingToolsSummary(processedRuns)

	if len(result) != 1 {
		t.Errorf("Expected 1 missing tool summary, got %d", len(result))
	}

	if len(result) > 0 {
		if result[0].WorkflowsDisplay == "" {
			t.Error("WorkflowsDisplay not populated")
		}
		if result[0].FirstReasonDisplay == "" {
			t.Error("FirstReasonDisplay not populated")
		}
	}
}

// TestBuildMCPFailuresSummaryPopulatesDisplay tests that display fields are populated
func TestBuildMCPFailuresSummaryPopulatesDisplay(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				WorkflowName: "test-workflow",
			},
			MCPFailures: []MCPFailureReport{
				{
					ServerName:   "github-mcp-server",
					WorkflowName: "test-workflow",
					RunID:        12345,
				},
			},
		},
	}

	result := buildMCPFailuresSummary(processedRuns)

	if len(result) != 1 {
		t.Errorf("Expected 1 MCP failure summary, got %d", len(result))
	}

	if len(result) > 0 {
		if result[0].WorkflowsDisplay == "" {
			t.Error("WorkflowsDisplay not populated")
		}
	}
}
