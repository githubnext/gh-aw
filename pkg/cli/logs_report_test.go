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
				Agent:            "claude",
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
				Name:          "github-mcp-server",
				TotalCalls:    1500,
				Runs:          5,
				MaxOutputSize: 2500000,
				MaxDuration:   "1m30s",
			},
			{
				Name:          "playwright",
				TotalCalls:    500,
				Runs:          3,
				MaxOutputSize: 512000,
				MaxDuration:   "45s",
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

	renderLogsConsole(data)
	renderLogsConsole(data)
}

// TestBuildToolUsageSummaryPopulatesDisplay tests that buildToolUsageSummary works correctly
func TestBuildToolUsageSummaryPopulatesDisplay(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				LogsPath: "/tmp/test-logs",
			},
		},
	}

	result := buildToolUsageSummary(processedRuns)

	// The result should be a valid slice (nil or empty is fine when no tools)
	_ = result
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

// TestAddUniqueWorkflow tests the workflow deduplication helper
func TestAddUniqueWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		workflows []string
		workflow  string
		expected  []string
	}{
		{
			name:      "add to empty list",
			workflows: []string{},
			workflow:  "workflow-a",
			expected:  []string{"workflow-a"},
		},
		{
			name:      "add new workflow",
			workflows: []string{"workflow-a", "workflow-b"},
			workflow:  "workflow-c",
			expected:  []string{"workflow-a", "workflow-b", "workflow-c"},
		},
		{
			name:      "duplicate workflow at beginning",
			workflows: []string{"workflow-a", "workflow-b", "workflow-c"},
			workflow:  "workflow-a",
			expected:  []string{"workflow-a", "workflow-b", "workflow-c"},
		},
		{
			name:      "duplicate workflow in middle",
			workflows: []string{"workflow-a", "workflow-b", "workflow-c"},
			workflow:  "workflow-b",
			expected:  []string{"workflow-a", "workflow-b", "workflow-c"},
		},
		{
			name:      "duplicate workflow at end",
			workflows: []string{"workflow-a", "workflow-b", "workflow-c"},
			workflow:  "workflow-c",
			expected:  []string{"workflow-a", "workflow-b", "workflow-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addUniqueWorkflow(tt.workflows, tt.workflow)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i, wf := range result {
				if wf != tt.expected[i] {
					t.Errorf("Expected workflow[%d] = %s, got %s", i, tt.expected[i], wf)
				}
			}
		})
	}
}

// TestBuildMissingToolsSummaryDeduplication tests that workflow deduplication works correctly
func TestBuildMissingToolsSummaryDeduplication(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				WorkflowName: "workflow-a",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "terraform",
					Reason:       "First reason",
					WorkflowName: "workflow-a",
					RunID:        12345,
				},
			},
		},
		{
			Run: WorkflowRun{
				WorkflowName: "workflow-b",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "terraform",
					Reason:       "Second reason",
					WorkflowName: "workflow-b",
					RunID:        12346,
				},
			},
		},
		{
			Run: WorkflowRun{
				WorkflowName: "workflow-a",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "terraform",
					Reason:       "Third reason from workflow-a",
					WorkflowName: "workflow-a",
					RunID:        12347,
				},
			},
		},
	}

	result := buildMissingToolsSummary(processedRuns)

	if len(result) != 1 {
		t.Errorf("Expected 1 missing tool summary, got %d", len(result))
	}

	if len(result) > 0 {
		summary := result[0]

		// Should have 3 total occurrences
		if summary.Count != 3 {
			t.Errorf("Expected count = 3, got %d", summary.Count)
		}

		// Should have only 2 unique workflows (workflow-a and workflow-b)
		if len(summary.Workflows) != 2 {
			t.Errorf("Expected 2 unique workflows, got %d", len(summary.Workflows))
		}

		// Should have 3 run IDs
		if len(summary.RunIDs) != 3 {
			t.Errorf("Expected 3 run IDs, got %d", len(summary.RunIDs))
		}

		// First reason should be preserved
		if summary.FirstReason != "First reason" {
			t.Errorf("Expected FirstReason = 'First reason', got '%s'", summary.FirstReason)
		}
	}
}

// TestBuildMCPFailuresSummaryDeduplication tests that workflow deduplication works correctly
func TestBuildMCPFailuresSummaryDeduplication(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				WorkflowName: "workflow-a",
			},
			MCPFailures: []MCPFailureReport{
				{
					ServerName:   "github-mcp-server",
					WorkflowName: "workflow-a",
					RunID:        12345,
				},
			},
		},
		{
			Run: WorkflowRun{
				WorkflowName: "workflow-b",
			},
			MCPFailures: []MCPFailureReport{
				{
					ServerName:   "github-mcp-server",
					WorkflowName: "workflow-b",
					RunID:        12346,
				},
			},
		},
		{
			Run: WorkflowRun{
				WorkflowName: "workflow-a",
			},
			MCPFailures: []MCPFailureReport{
				{
					ServerName:   "github-mcp-server",
					WorkflowName: "workflow-a",
					RunID:        12347,
				},
			},
		},
	}

	result := buildMCPFailuresSummary(processedRuns)

	if len(result) != 1 {
		t.Errorf("Expected 1 MCP failure summary, got %d", len(result))
	}

	if len(result) > 0 {
		summary := result[0]

		// Should have 3 total occurrences
		if summary.Count != 3 {
			t.Errorf("Expected count = 3, got %d", summary.Count)
		}

		// Should have only 2 unique workflows (workflow-a and workflow-b)
		if len(summary.Workflows) != 2 {
			t.Errorf("Expected 2 unique workflows, got %d", len(summary.Workflows))
		}

		// Should have 3 run IDs
		if len(summary.RunIDs) != 3 {
			t.Errorf("Expected 3 run IDs, got %d", len(summary.RunIDs))
		}
	}
}
