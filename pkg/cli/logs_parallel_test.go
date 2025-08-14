package cli

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDownloadRunArtifactsParallel(t *testing.T) {
	// Test with empty runs slice
	results := downloadRunArtifactsConcurrent([]WorkflowRun{}, "./test-logs", false, 5)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty runs, got %d", len(results))
	}

	// Test with mock runs
	runs := []WorkflowRun{
		{
			DatabaseID:   12345,
			Number:       1,
			Status:       "completed",
			Conclusion:   "success",
			WorkflowName: "Test Workflow",
			CreatedAt:    time.Now().Add(-1 * time.Hour),
			StartedAt:    time.Now().Add(-55 * time.Minute),
			UpdatedAt:    time.Now().Add(-50 * time.Minute),
		},
		{
			DatabaseID:   12346,
			Number:       2,
			Status:       "completed",
			Conclusion:   "failure",
			WorkflowName: "Test Workflow",
			CreatedAt:    time.Now().Add(-2 * time.Hour),
			StartedAt:    time.Now().Add(-115 * time.Minute),
			UpdatedAt:    time.Now().Add(-110 * time.Minute),
		},
	}

	// This will fail since we don't have real GitHub CLI access,
	// but we can verify the structure and that no panics occur
	results = downloadRunArtifactsConcurrent(runs, "./test-logs", false, 5)

	// We expect 2 results even if they fail
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify we have results for all our runs (order may vary due to parallel execution)
	foundRuns := make(map[int64]bool)
	for _, result := range results {
		foundRuns[result.Run.DatabaseID] = true

		// Verify the LogsPath follows the expected pattern (normalize path separators)
		expectedSuffix := fmt.Sprintf("run-%d", result.Run.DatabaseID)
		if !strings.Contains(result.LogsPath, expectedSuffix) {
			t.Errorf("Expected LogsPath to contain %s, got %s", expectedSuffix, result.LogsPath)
		}
	}

	// Verify we processed all the runs we sent
	for _, run := range runs {
		if !foundRuns[run.DatabaseID] {
			t.Errorf("Missing result for run %d", run.DatabaseID)
		}
	}
}

func TestDownloadRunArtifactsParallelMaxRuns(t *testing.T) {
	// Test maxRuns parameter
	runs := []WorkflowRun{
		{DatabaseID: 1, Status: "completed"},
		{DatabaseID: 2, Status: "completed"},
		{DatabaseID: 3, Status: "completed"},
		{DatabaseID: 4, Status: "completed"},
		{DatabaseID: 5, Status: "completed"},
	}

	// Limit to 3 runs
	results := downloadRunArtifactsConcurrent(runs, "./test-logs", false, 3)

	if len(results) != 3 {
		t.Errorf("Expected 3 results when maxRuns=3, got %d", len(results))
	}

	// Verify we got exactly 3 results from the first 3 runs (order may vary due to parallel execution)
	expectedIDs := map[int64]bool{1: false, 2: false, 3: false}
	for _, result := range results {
		if _, expected := expectedIDs[result.Run.DatabaseID]; expected {
			expectedIDs[result.Run.DatabaseID] = true
		} else {
			t.Errorf("Got unexpected DatabaseID %d", result.Run.DatabaseID)
		}
	}

	// Verify all expected IDs were found
	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Missing expected DatabaseID %d", id)
		}
	}
}

func TestDownloadResult(t *testing.T) {
	// Test DownloadResult structure
	run := WorkflowRun{
		DatabaseID: 12345,
		Status:     "completed",
	}

	result := DownloadResult{
		Run:      run,
		LogsPath: "./test-path",
		Skipped:  false,
		Error:    nil,
	}

	if result.Run.DatabaseID != 12345 {
		t.Errorf("Expected DatabaseID 12345, got %d", result.Run.DatabaseID)
	}

	if result.LogsPath != "./test-path" {
		t.Errorf("Expected LogsPath './test-path', got %s", result.LogsPath)
	}

	if result.Skipped {
		t.Error("Expected Skipped to be false")
	}

	if result.Error != nil {
		t.Errorf("Expected Error to be nil, got %v", result.Error)
	}
}

func TestMaxConcurrentDownloads(t *testing.T) {
	// Test that MaxConcurrentDownloads constant is properly defined
	if MaxConcurrentDownloads <= 0 {
		t.Errorf("MaxConcurrentDownloads should be positive, got %d", MaxConcurrentDownloads)
	}

	if MaxConcurrentDownloads > 20 {
		t.Errorf("MaxConcurrentDownloads should be reasonable (<=20), got %d", MaxConcurrentDownloads)
	}
}
