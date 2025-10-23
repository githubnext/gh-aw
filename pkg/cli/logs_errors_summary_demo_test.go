package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestErrorsSummaryDemo demonstrates the new error summary functionality
func TestErrorsSummaryDemo(t *testing.T) {
	// Create mock processed runs with simulated errors
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   12345,
				WorkflowName: "test-workflow",
				URL:          "https://github.com/test/repo/actions/runs/12345",
				LogsPath:     "/tmp/test-logs/run-12345",
			},
		},
	}

	// Build error summaries
	errorsSummary, warningsSummary := buildErrorsSummary(processedRuns)

	// Verify the function works correctly with empty data
	if len(errorsSummary) != 0 {
		t.Errorf("Expected 0 errors in summary with empty logs, got %d", len(errorsSummary))
	}

	if len(warningsSummary) != 0 {
		t.Errorf("Expected 0 warnings in summary with empty logs, got %d", len(warningsSummary))
	}

	// Test the structure of ErrorSummary
	demoSummary := ErrorSummary{
		Message:      "Permission denied: Unable to access resource",
		Count:        15,
		PatternID:    "common-generic-error",
		Engine:       "copilot",
		RunID:        12345,
		RunURL:       "https://github.com/test/repo/actions/runs/12345",
		WorkflowName: "test-workflow",
	}

	// Verify all fields are accessible
	if demoSummary.Message == "" {
		t.Error("Message field should not be empty")
	}
	if demoSummary.Count != 15 {
		t.Errorf("Expected count 15, got %d", demoSummary.Count)
	}
	if demoSummary.PatternID != "common-generic-error" {
		t.Errorf("Expected pattern ID 'common-generic-error', got %s", demoSummary.PatternID)
	}

	t.Logf("Demo ErrorSummary structure: %+v", demoSummary)
}

// TestLogsDataWithErrorSummaries demonstrates the complete logs data structure
func TestLogsDataWithErrorSummaries(t *testing.T) {
	// Create a complete LogsData structure with error summaries
	logsData := LogsData{
		Summary: LogsSummary{
			TotalRuns:    5,
			TotalErrors:  25,
			TotalWarnings: 10,
		},
		ErrorsSummary: []ErrorSummary{
			{
				Message:      "Authentication failed",
				Count:        15,
				PatternID:    "common-generic-error",
				Engine:       "copilot",
				RunID:        12345,
				RunURL:       "https://github.com/test/repo/actions/runs/12345",
				WorkflowName: "auth-workflow",
			},
			{
				Message:      "Network timeout",
				Count:        10,
				PatternID:    "common-generic-error",
				Engine:       "claude",
				RunID:        12346,
				RunURL:       "https://github.com/test/repo/actions/runs/12346",
				WorkflowName: "network-workflow",
			},
		},
		WarningsSummary: []ErrorSummary{
			{
				Message:      "Rate limit approaching",
				Count:        8,
				PatternID:    "common-generic-warning",
				Engine:       "copilot",
				RunID:        12347,
				RunURL:       "https://github.com/test/repo/actions/runs/12347",
				WorkflowName: "api-workflow",
			},
		},
		LogsLocation: filepath.Join("/tmp", "logs"),
	}

	// Verify the structure can be marshalled to JSON
	jsonData, err := json.MarshalIndent(logsData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal LogsData to JSON: %v", err)
	}

	t.Logf("LogsData JSON structure:\n%s", string(jsonData))

	// Verify the summaries are sorted by count
	if len(logsData.ErrorsSummary) >= 2 {
		if logsData.ErrorsSummary[0].Count < logsData.ErrorsSummary[1].Count {
			t.Error("Error summaries should be sorted by count in descending order")
		}
	}

	// Verify all required fields are present
	if len(logsData.ErrorsSummary) > 0 {
		firstError := logsData.ErrorsSummary[0]
		if firstError.Message == "" {
			t.Error("Error summary message should not be empty")
		}
		if firstError.PatternID == "" {
			t.Error("Error summary pattern ID should not be empty")
		}
		if firstError.RunID == 0 {
			t.Error("Error summary run ID should not be zero")
		}
	}
}

// TestErrorPatternIDsAreUnique verifies all error patterns have unique IDs
func TestErrorPatternIDsAreUnique(t *testing.T) {
	// Get all error patterns from common patterns
	commonPatterns := workflow.GetCommonErrorPatterns()

	// Track pattern IDs
	seenIDs := make(map[string]bool)
	duplicates := []string{}

	for _, pattern := range commonPatterns {
		if pattern.ID == "" {
			t.Errorf("Pattern '%s' has no ID", pattern.Description)
			continue
		}

		if seenIDs[pattern.ID] {
			duplicates = append(duplicates, pattern.ID)
		} else {
			seenIDs[pattern.ID] = true
		}
	}

	if len(duplicates) > 0 {
		t.Errorf("Found duplicate pattern IDs: %v", duplicates)
	}

	t.Logf("Verified %d unique error pattern IDs", len(seenIDs))
}
