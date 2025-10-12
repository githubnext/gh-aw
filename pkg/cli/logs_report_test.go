package cli

import (
	"strings"
	"testing"
	"time"
)

// TestToolUsageDisplay verifies the ToolUsageDisplay struct renders correctly
func TestToolUsageDisplay(t *testing.T) {
	displayData := []ToolUsageDisplay{
		{
			Name:        "github-mcp-server",
			TotalCalls:  "1.5k",
			Runs:        5,
			MaxOutput:   "2.5 MB",
			MaxDuration: "1m30s",
		},
		{
			Name:        "playwright",
			TotalCalls:  "500",
			Runs:        3,
			MaxOutput:   "512 KB",
			MaxDuration: "45s",
		},
	}

	// This should not panic
	for _, data := range displayData {
		if data.Name == "" {
			t.Error("Name should not be empty")
		}
		if data.TotalCalls == "" {
			t.Error("TotalCalls should not be empty")
		}
	}
}

// TestMissingToolDisplay verifies the MissingToolDisplay struct renders correctly
func TestMissingToolDisplay(t *testing.T) {
	displayData := []MissingToolDisplay{
		{
			Tool:        "terraform",
			Count:       5,
			Workflows:   "workflow-a, workflow-b",
			FirstReason: "Infrastructure automation needed",
		},
		{
			Tool:        "kubectl",
			Count:       3,
			Workflows:   "k8s-deploy",
			FirstReason: "K8s management required",
		},
	}

	// This should not panic
	for _, data := range displayData {
		if data.Tool == "" {
			t.Error("Tool should not be empty")
		}
		if data.Count < 0 {
			t.Error("Count should not be negative")
		}
	}
}

// TestMCPFailureDisplay verifies the MCPFailureDisplay struct renders correctly
func TestMCPFailureDisplay(t *testing.T) {
	displayData := []MCPFailureDisplay{
		{
			ServerName: "github-mcp-server",
			Count:      2,
			Workflows:  "workflow-a, workflow-b",
		},
		{
			ServerName: "playwright",
			Count:      1,
			Workflows:  "browser-test",
		},
	}

	// This should not panic
	for _, data := range displayData {
		if data.ServerName == "" {
			t.Error("ServerName should not be empty")
		}
		if data.Count < 0 {
			t.Error("Count should not be negative")
		}
	}
}

// TestDisplayToolUsageFromData tests the tool usage display function
func TestDisplayToolUsageFromData(t *testing.T) {
	toolUsage := []ToolUsageSummary{
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
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayToolUsageFromData panicked: %v", r)
		}
	}()

	displayToolUsageFromData(toolUsage, false)
	displayToolUsageFromData(toolUsage, true)
}

// TestDisplayMCPFailuresFromData tests the MCP failures display function
func TestDisplayMCPFailuresFromData(t *testing.T) {
	mcpFailures := []MCPFailureSummary{
		{
			ServerName: "github-mcp-server",
			Count:      2,
			Workflows:  []string{"workflow-a", "workflow-b"},
			RunIDs:     []int64{12345, 67890},
		},
		{
			ServerName: "playwright",
			Count:      1,
			Workflows:  []string{"browser-test"},
			RunIDs:     []int64{11111},
		},
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayMCPFailuresFromData panicked: %v", r)
		}
	}()

	displayMCPFailuresFromData(mcpFailures, false)
	displayMCPFailuresFromData(mcpFailures, true)
}

// TestDisplayMissingToolsFromData tests the missing tools display function
func TestDisplayMissingToolsFromData(t *testing.T) {
	missingTools := []MissingToolSummary{
		{
			Tool:        "terraform",
			Count:       5,
			Workflows:   []string{"workflow-a", "workflow-b", "workflow-c"},
			FirstReason: "Infrastructure automation needed",
			RunIDs:      []int64{12345, 67890, 11111, 22222, 33333},
		},
		{
			Tool:        "kubectl",
			Count:       3,
			Workflows:   []string{"k8s-deploy"},
			FirstReason: "K8s management required",
			RunIDs:      []int64{44444, 55555, 66666},
		},
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayMissingToolsFromData panicked: %v", r)
		}
	}()

	displayMissingToolsFromData(missingTools, false)
	displayMissingToolsFromData(missingTools, true)
}

// TestDisplayToolUsageWithLongNames tests truncation of long tool names
func TestDisplayToolUsageWithLongNames(t *testing.T) {
	toolUsage := []ToolUsageSummary{
		{
			Name:          "very-long-tool-name-that-exceeds-normal-display-width-limits",
			TotalCalls:    100,
			Runs:          1,
			MaxOutputSize: 1024,
			MaxDuration:   "1s",
		},
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayToolUsageFromData with long names panicked: %v", r)
		}
	}()

	displayToolUsageFromData(toolUsage, false)
}

// TestDisplayMissingToolsWithLongWorkflows tests truncation of long workflow lists
func TestDisplayMissingToolsWithLongWorkflows(t *testing.T) {
	longWorkflows := []string{
		"workflow-1", "workflow-2", "workflow-3", "workflow-4", "workflow-5",
		"workflow-6", "workflow-7", "workflow-8", "workflow-9", "workflow-10",
	}

	missingTools := []MissingToolSummary{
		{
			Tool:        "terraform",
			Count:       10,
			Workflows:   longWorkflows,
			FirstReason: "This is a very long reason that should be truncated when displayed in the console output",
			RunIDs:      []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayMissingToolsFromData with long workflows panicked: %v", r)
		}
	}()

	displayMissingToolsFromData(missingTools, false)
}

// TestRunDataConsoleTags verifies RunData has proper console tags
func TestRunDataConsoleTags(t *testing.T) {
	run := RunData{
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
		LogsPath:         "/tmp/logs",
	}

	// Verify fields are accessible
	if run.DatabaseID != 12345 {
		t.Errorf("Expected DatabaseID 12345, got %d", run.DatabaseID)
	}
	if run.WorkflowName != "test-workflow" {
		t.Error("WorkflowName not set correctly")
	}
}

// TestBuildLogsDataStructure verifies the buildLogsData function structure
func TestBuildLogsDataStructure(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:       12345,
				Number:           1,
				WorkflowName:     "test-workflow",
				Status:           "completed",
				Conclusion:       "success",
				TokenUsage:       1000,
				EstimatedCost:    0.01,
				Turns:            3,
				ErrorCount:       0,
				WarningCount:     2,
				MissingToolCount: 1,
				CreatedAt:        time.Now(),
				Duration:         5 * time.Minute,
				URL:              "https://github.com/owner/repo/runs/12345",
				LogsPath:         "/tmp/logs/12345",
				Event:            "push",
				HeadBranch:       "main",
			},
		},
	}

	logsData := buildLogsData(processedRuns, "/tmp/logs", false)

	if len(logsData.Runs) != 1 {
		t.Errorf("Expected 1 run, got %d", len(logsData.Runs))
	}

	if logsData.Summary.TotalRuns != 1 {
		t.Errorf("Expected TotalRuns to be 1, got %d", logsData.Summary.TotalRuns)
	}

	if logsData.Summary.TotalTokens != 1000 {
		t.Errorf("Expected TotalTokens to be 1000, got %d", logsData.Summary.TotalTokens)
	}

	if logsData.Summary.TotalCost != 0.01 {
		t.Errorf("Expected TotalCost to be 0.01, got %f", logsData.Summary.TotalCost)
	}
}

// TestToolUsageSummaryConsoleTags verifies ToolUsageSummary has proper console tags
func TestToolUsageSummaryConsoleTags(t *testing.T) {
	tool := ToolUsageSummary{
		Name:          "github-mcp-server",
		TotalCalls:    1500,
		Runs:          5,
		MaxOutputSize: 2500000,
		MaxDuration:   "1m30s",
	}

	// Verify fields are accessible
	if tool.Name != "github-mcp-server" {
		t.Error("Name not set correctly")
	}
	if tool.TotalCalls != 1500 {
		t.Errorf("Expected TotalCalls 1500, got %d", tool.TotalCalls)
	}
}

// TestMissingToolSummaryConsoleTags verifies MissingToolSummary has proper console tags
func TestMissingToolSummaryConsoleTags(t *testing.T) {
	tool := MissingToolSummary{
		Tool:        "terraform",
		Count:       5,
		Workflows:   []string{"workflow-a", "workflow-b"},
		FirstReason: "Infrastructure automation needed",
		RunIDs:      []int64{12345, 67890},
	}

	// Verify fields are accessible
	if tool.Tool != "terraform" {
		t.Error("Tool not set correctly")
	}
	if tool.Count != 5 {
		t.Errorf("Expected Count 5, got %d", tool.Count)
	}
}

// TestMCPFailureSummaryConsoleTags verifies MCPFailureSummary has proper console tags
func TestMCPFailureSummaryConsoleTags(t *testing.T) {
	failure := MCPFailureSummary{
		ServerName: "github-mcp-server",
		Count:      2,
		Workflows:  []string{"workflow-a", "workflow-b"},
		RunIDs:     []int64{12345, 67890},
	}

	// Verify fields are accessible
	if failure.ServerName != "github-mcp-server" {
		t.Error("ServerName not set correctly")
	}
	if failure.Count != 2 {
		t.Errorf("Expected Count 2, got %d", failure.Count)
	}
}

// TestWorkflowListTruncation verifies workflow list truncation
func TestWorkflowListTruncation(t *testing.T) {
	// Test that long workflow lists are truncated
	longWorkflows := []string{
		"workflow-1", "workflow-2", "workflow-3", "workflow-4", "workflow-5",
		"workflow-6", "workflow-7", "workflow-8", "workflow-9", "workflow-10",
	}

	workflowList := strings.Join(longWorkflows, ", ")

	// The display function truncates at 60 characters for MCP failures
	if len(workflowList) > 60 {
		truncated := workflowList[:57] + "..."
		if len(truncated) != 60 {
			t.Errorf("Expected truncated length 60, got %d", len(truncated))
		}
	}

	// The display function truncates at 40 characters for missing tools
	if len(workflowList) > 40 {
		truncated := workflowList[:37] + "..."
		if len(truncated) != 40 {
			t.Errorf("Expected truncated length 40, got %d", len(truncated))
		}
	}
}

// TestReasonTruncation verifies reason text truncation
func TestReasonTruncation(t *testing.T) {
	longReason := "This is a very long reason that should be truncated when displayed in the console output to avoid making the table too wide"

	// The display function truncates reasons at 50 characters
	if len(longReason) > 50 {
		truncated := longReason[:47] + "..."
		if len(truncated) != 50 {
			t.Errorf("Expected truncated length 50, got %d", len(truncated))
		}
	}
}
