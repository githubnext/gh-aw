package cli

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestBuildErrorsSummary(t *testing.T) {
	// Create test runs with some errors and warnings
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   123,
				WorkflowName: "test-workflow-1",
				URL:          "https://github.com/test/repo/actions/runs/123",
				LogsPath:     "/logs/run-123",
			},
		},
		{
			Run: WorkflowRun{
				DatabaseID:   456,
				WorkflowName: "test-workflow-2",
				URL:          "https://github.com/test/repo/actions/runs/456",
				LogsPath:     "/logs/run-456",
			},
		},
	}

	// Manually add metrics to the runs by creating a mock ExtractLogMetricsFromRun
	// We'll test with empty metrics for now to ensure the function doesn't crash
	errorsSummary, warningsSummary := buildErrorsSummary(processedRuns)

	// With no actual log metrics, we should get empty summaries
	if len(errorsSummary) != 0 {
		t.Errorf("Expected 0 errors in summary, got %d", len(errorsSummary))
	}

	if len(warningsSummary) != 0 {
		t.Errorf("Expected 0 warnings in summary, got %d", len(warningsSummary))
	}
}

func TestErrorPatternHasID(t *testing.T) {
	// Test that error patterns have IDs
	patterns := workflow.GetCommonErrorPatterns()

	if len(patterns) == 0 {
		t.Fatal("Expected common error patterns, got none")
	}

	for i, pattern := range patterns {
		if pattern.ID == "" {
			t.Errorf("Pattern %d (%s) has no ID", i, pattern.Description)
		}
		if pattern.Pattern == "" {
			t.Errorf("Pattern %d has no pattern", i)
		}
	}
}

func TestErrorSummaryStructure(t *testing.T) {
	// Test the ErrorSummary struct has expected fields
	summary := ErrorSummary{
		Message:      "Test error message",
		Count:        5,
		PatternID:    "test-pattern-id",
		Engine:       "copilot",
		RunID:        123456,
		RunURL:       "https://github.com/test/repo/actions/runs/123456",
		WorkflowName: "test-workflow",
	}

	if summary.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got %s", summary.Message)
	}
	if summary.Count != 5 {
		t.Errorf("Expected count 5, got %d", summary.Count)
	}
	if summary.PatternID != "test-pattern-id" {
		t.Errorf("Expected pattern ID 'test-pattern-id', got %s", summary.PatternID)
	}
	if summary.Engine != "copilot" {
		t.Errorf("Expected engine 'copilot', got %s", summary.Engine)
	}
}
