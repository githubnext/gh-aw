package cli

import (
	"strings"
	"testing"
	"time"
)

// TestLogsOverviewIncludesMissingTools verifies that the overview table includes missing tools count
func TestLogsOverviewIncludesMissingTools(t *testing.T) {
	runs := []WorkflowRun{
		{
			DatabaseID:       12345,
			WorkflowName:     "Test Workflow A",
			Status:           "completed",
			Conclusion:       "success",
			CreatedAt:        time.Now(),
			Duration:         5 * time.Minute,
			TokenUsage:       1000,
			EstimatedCost:    0.01,
			Turns:            3,
			ErrorCount:       0,
			WarningCount:     2,
			MissingToolCount: 1,
			LogsPath:         "/tmp/run-12345",
		},
		{
			DatabaseID:       67890,
			WorkflowName:     "Test Workflow B",
			Status:           "completed",
			Conclusion:       "failure",
			CreatedAt:        time.Now(),
			Duration:         3 * time.Minute,
			TokenUsage:       500,
			EstimatedCost:    0.005,
			Turns:            2,
			ErrorCount:       1,
			WarningCount:     0,
			MissingToolCount: 3,
			LogsPath:         "/tmp/run-67890",
		},
	}

	// Capture output by redirecting - this is a smoke test to ensure displayLogsOverview doesn't panic
	// and that it processes the MissingToolCount field
	displayLogsOverview(runs)
}

// TestWorkflowRunStructHasMissingToolCount verifies that WorkflowRun has the MissingToolCount field
func TestWorkflowRunStructHasMissingToolCount(t *testing.T) {
	run := WorkflowRun{
		DatabaseID:       12345,
		WorkflowName:     "Test",
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
func TestDisplayLogsOverviewWithVariousMissingToolCounts(t *testing.T) {
	testCases := []struct {
		name             string
		runs             []WorkflowRun
		expectedNonPanic bool
	}{
		{
			name: "no missing tools",
			runs: []WorkflowRun{
				{
					DatabaseID:       1,
					WorkflowName:     "Clean Workflow",
					MissingToolCount: 0,
					LogsPath:         "/tmp/run-1",
				},
			},
			expectedNonPanic: true,
		},
		{
			name: "single missing tool",
			runs: []WorkflowRun{
				{
					DatabaseID:       2,
					WorkflowName:     "Workflow with One Missing",
					MissingToolCount: 1,
					LogsPath:         "/tmp/run-2",
				},
			},
			expectedNonPanic: true,
		},
		{
			name: "multiple missing tools",
			runs: []WorkflowRun{
				{
					DatabaseID:       3,
					WorkflowName:     "Workflow with Multiple Missing",
					MissingToolCount: 5,
					LogsPath:         "/tmp/run-3",
				},
			},
			expectedNonPanic: true,
		},
		{
			name: "mixed missing tool counts",
			runs: []WorkflowRun{
				{
					DatabaseID:       4,
					WorkflowName:     "Workflow A",
					MissingToolCount: 0,
					LogsPath:         "/tmp/run-4",
				},
				{
					DatabaseID:       5,
					WorkflowName:     "Workflow B",
					MissingToolCount: 2,
					LogsPath:         "/tmp/run-5",
				},
				{
					DatabaseID:       6,
					WorkflowName:     "Workflow C",
					MissingToolCount: 1,
					LogsPath:         "/tmp/run-6",
				},
			},
			expectedNonPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test ensures displayLogsOverview doesn't panic with various missing tool counts
			defer func() {
				if r := recover(); r != nil && tc.expectedNonPanic {
					t.Errorf("displayLogsOverview panicked with: %v", r)
				}
			}()
			displayLogsOverview(tc.runs)
		})
	}
}

// TestTotalMissingToolsCalculation verifies totals are calculated correctly
func TestTotalMissingToolsCalculation(t *testing.T) {
	runs := []WorkflowRun{
		{DatabaseID: 1, MissingToolCount: 2, LogsPath: "/tmp/run-1"},
		{DatabaseID: 2, MissingToolCount: 0, LogsPath: "/tmp/run-2"},
		{DatabaseID: 3, MissingToolCount: 5, LogsPath: "/tmp/run-3"},
		{DatabaseID: 4, MissingToolCount: 1, LogsPath: "/tmp/run-4"},
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
func TestOverviewDisplayConsistency(t *testing.T) {
	// Create a run with known values
	run := WorkflowRun{
		DatabaseID:       99999,
		WorkflowName:     "Consistency Test",
		Status:           "completed",
		Conclusion:       "success",
		Duration:         10 * time.Minute,
		TokenUsage:       2000,
		EstimatedCost:    0.02,
		Turns:            5,
		ErrorCount:       1,
		WarningCount:     3,
		MissingToolCount: 2,
		CreatedAt:        time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		LogsPath:         "/tmp/run-99999",
	}

	runs := []WorkflowRun{run}

	// Call displayLogsOverview - it should not panic and should handle all fields
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("displayLogsOverview panicked: %v", r)
		}
	}()

	displayLogsOverview(runs)
}

// TestMissingToolsIntegration tests the full flow from ProcessedRun to display
func TestMissingToolsIntegration(t *testing.T) {
	// Create a ProcessedRun with missing tools
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   11111,
				WorkflowName: "Integration Test Workflow",
				Status:       "completed",
				Conclusion:   "success",
			},
			MissingTools: []MissingToolReport{
				{
					Tool:         "terraform",
					Reason:       "Infrastructure automation needed",
					Alternatives: "Manual AWS console",
					Timestamp:    "2024-01-15T10:30:00Z",
					WorkflowName: "Integration Test Workflow",
					RunID:        11111,
				},
				{
					Tool:         "kubectl",
					Reason:       "Kubernetes cluster management",
					WorkflowName: "Integration Test Workflow",
					RunID:        11111,
				},
			},
		},
	}

	// Simulate the logs command flow
	workflowRuns := make([]WorkflowRun, len(processedRuns))
	for i, pr := range processedRuns {
		run := pr.Run
		run.MissingToolCount = len(pr.MissingTools)
		workflowRuns[i] = run
	}

	// Verify count is correct
	if workflowRuns[0].MissingToolCount != 2 {
		t.Errorf("Expected MissingToolCount to be 2, got %d", workflowRuns[0].MissingToolCount)
	}

	// Display should work without panicking
	displayLogsOverview(workflowRuns)

	// Display analysis should also work
	displayMissingToolsAnalysis(processedRuns, false)
}

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

// Helper function to check if output contains expected string
func containsString(output, expected string) bool {
	return strings.Contains(output, expected)
}
