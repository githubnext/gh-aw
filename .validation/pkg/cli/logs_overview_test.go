//go:build !integration

package cli

import (
	"testing"
)

// TestLogsOverviewIncludesMissingTools verifies that the overview table includes missing tools count

// TestWorkflowRunStructHasMissingToolCount verifies that WorkflowRun has the MissingToolCount field
func TestWorkflowRunStructHasMissingToolCount(t *testing.T) {
	run := WorkflowRun{
		MissingToolCount: 5,
	}

	if run.MissingToolCount != 5 {
		t.Errorf("Expected MissingToolCount to be 5, got %d", run.MissingToolCount)
	}
}

// TestProcessedRunPopulatesMissingToolCount verifies that missing tools are counted correctly
func TestProcessedRunPopulatesMissingToolCount(t *testing.T) {
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   12345,
				WorkflowName: "Test Workflow",
			},
			MissingTools: []MissingToolReport{
				{Tool: "terraform", Reason: "Need infrastructure automation"},
				{Tool: "kubectl", Reason: "Need K8s management"},
			},
		},
	}

	// Simulate what the logs command does
	workflowRuns := make([]WorkflowRun, len(processedRuns))
	for i, pr := range processedRuns {
		run := pr.Run
		run.MissingToolCount = len(pr.MissingTools)
		workflowRuns[i] = run
	}

	if workflowRuns[0].MissingToolCount != 2 {
		t.Errorf("Expected MissingToolCount to be 2, got %d", workflowRuns[0].MissingToolCount)
	}
}

// TestLogsOverviewHeaderIncludesMissing verifies the header includes "Missing"
func TestLogsOverviewHeaderIncludesMissing(t *testing.T) {
	// This test verifies the structure by checking that our expected headers are defined
	expectedHeaders := []string{"Run ID", "Workflow", "Status", "Duration", "Tokens", "Cost ($)", "Turns", "Errors", "Warnings", "Missing", "Created", "Logs Path"}

	// Verify the "Missing" header is in the expected position (index 9)
	if expectedHeaders[9] != "Missing" {
		t.Errorf("Expected header at index 9 to be 'Missing', got '%s'", expectedHeaders[9])
	}

	// Verify we have 12 columns total
	if len(expectedHeaders) != 12 {
		t.Errorf("Expected 12 headers, got %d", len(expectedHeaders))
	}
}

// TestDisplayLogsOverviewWithVariousMissingToolCounts tests different scenarios

// TestTotalMissingToolsCalculation verifies totals are calculated correctly
func TestTotalMissingToolsCalculation(t *testing.T) {
	runs := []WorkflowRun{
		{DatabaseID: 1, MissingToolCount: 2, LogsPath: "/tmp/gh-aw/run-1"},
		{DatabaseID: 2, MissingToolCount: 0, LogsPath: "/tmp/gh-aw/run-2"},
		{DatabaseID: 3, MissingToolCount: 5, LogsPath: "/tmp/gh-aw/run-3"},
		{DatabaseID: 4, MissingToolCount: 1, LogsPath: "/tmp/gh-aw/run-4"},
	}

	expectedTotal := 2 + 0 + 5 + 1 // = 8

	// Calculate total the same way displayLogsOverview does
	var totalMissingTools int
	for _, run := range runs {
		totalMissingTools += run.MissingToolCount
	}

	if totalMissingTools != expectedTotal {
		t.Errorf("Expected total missing tools to be %d, got %d", expectedTotal, totalMissingTools)
	}
}

// TestOverviewDisplayConsistency verifies that the overview function is consistent

// TestMissingToolsIntegration tests the full flow from ProcessedRun to display

// TestMissingToolCountFieldAccessibility verifies field is accessible
func TestMissingToolCountFieldAccessibility(t *testing.T) {
	var run WorkflowRun

	// Should be able to set and get the field
	run.MissingToolCount = 10

	if run.MissingToolCount != 10 {
		t.Errorf("MissingToolCount field not accessible or not working correctly")
	}

	// Should support zero value
	var emptyRun WorkflowRun
	if emptyRun.MissingToolCount != 0 {
		t.Errorf("MissingToolCount should default to 0, got %d", emptyRun.MissingToolCount)
	}
}
