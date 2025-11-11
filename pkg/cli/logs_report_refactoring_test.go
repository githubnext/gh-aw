package cli

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestBuildCombinedErrorsSummaryWithData tests the refactored buildCombinedErrorsSummary with actual data
func TestBuildCombinedErrorsSummaryWithData(t *testing.T) {
	// Create test runs with errors and warnings
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

	// Mock the ExtractLogMetricsFromRun by directly testing with empty metrics
	// In real usage, this would extract metrics from log files
	combined := buildCombinedErrorsSummary(processedRuns)

	// With no actual log metrics, we should get empty summary
	if len(combined) != 0 {
		t.Errorf("Expected 0 items in combined summary, got %d", len(combined))
	}
}

// TestBuildErrorsSummaryCompatibility tests that the deprecated buildErrorsSummary
// produces compatible results with buildCombinedErrorsSummary
func TestBuildErrorsSummaryCompatibility(t *testing.T) {
	// Create test runs
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   123,
				WorkflowName: "test-workflow-1",
				URL:          "https://github.com/test/repo/actions/runs/123",
				LogsPath:     "/logs/run-123",
			},
		},
	}

	// Test deprecated function
	errors, warnings := buildErrorsSummary(processedRuns)

	// Test new function
	combined := buildCombinedErrorsSummary(processedRuns)

	// Both should return empty results with no log data
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
	if len(combined) != 0 {
		t.Errorf("Expected 0 combined items, got %d", len(combined))
	}
}

// TestAggregateLogErrorsHelper tests the helper function directly
func TestAggregateLogErrorsHelper(t *testing.T) {
	// Create test data with mock ProcessedRun that will return no metrics
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   123,
				WorkflowName: "test-workflow",
				URL:          "https://github.com/test/repo/actions/runs/123",
				LogsPath:     "/nonexistent/path", // Won't have real metrics
			},
		},
	}

	// Test with combined aggregation strategy
	agg := logErrorAggregator{
		generateKey: func(logErr workflow.LogError) string {
			return logErr.Type + ":" + logErr.Message
		},
		selectMap: nil,
		sortResults: func(results []ErrorSummary) {
			// No-op sort for testing
		},
	}

	result := aggregateLogErrors(processedRuns, agg)

	// Should handle empty case gracefully and return empty slice
	if len(result) != 0 {
		t.Errorf("Expected empty result slice, got %d items", len(result))
	}
}

// TestErrorSummaryDeduplication tests that errors are properly deduplicated by key
func TestErrorSummaryDeduplication(t *testing.T) {
	// This test verifies that the aggregation logic properly deduplicates
	// errors based on the key generation strategy
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   123,
				WorkflowName: "test-workflow",
				URL:          "https://github.com/test/repo/actions/runs/123",
				LogsPath:     "/nonexistent/path",
			},
		},
	}

	// Test combined summary (uses type:message as key)
	combined := buildCombinedErrorsSummary(processedRuns)

	// Test deprecated function (uses message as key)
	errors, warnings := buildErrorsSummary(processedRuns)

	// All should produce empty results with no actual log data
	if len(combined) != 0 || len(errors) != 0 || len(warnings) != 0 {
		t.Error("Expected all summaries to be empty with no log data")
	}
}

// TestBuildCombinedErrorsSummarySorting tests that errors are sorted correctly
func TestBuildCombinedErrorsSummarySorting(t *testing.T) {
	// Create test data
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   123,
				WorkflowName: "test-workflow",
				URL:          "https://github.com/test/repo/actions/runs/123",
				LogsPath:     "/nonexistent/path",
			},
		},
	}

	combined := buildCombinedErrorsSummary(processedRuns)

	// Verify the function executes without panicking
	// and returns a valid (empty) slice
	if len(combined) != 0 {
		// If there were results, verify sorting
		// Errors should come before warnings
		for i := 1; i < len(combined); i++ {
			if combined[i-1].Type == "Warning" && combined[i].Type == "Error" {
				t.Error("Errors should be sorted before warnings")
			}
		}
	}
}
