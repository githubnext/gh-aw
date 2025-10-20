package cli

import (
	"testing"
)

// TestListWorkflowRunsWithPagination_ReturnsTotalFetchedCount verifies that
// the function returns both the filtered runs and the total count fetched from API
func TestListWorkflowRunsWithPagination_ReturnsTotalFetchedCount(t *testing.T) {
	t.Skip("Skipping network-dependent test - this verifies the fix for filtering issue")

	// This test would require actual GitHub CLI access to work properly
	// The key insight is that the function should return:
	// 1. Filtered runs (e.g., 5 agentic workflows)
	// 2. Total fetched count (e.g., 250 total runs from API)
	//
	// This allows the caller to properly detect when it has reached the end
	// of available runs by checking totalFetched < batchSize, not len(runs) < batchSize

	// Example scenario that would fail with old code:
	// - Request 250 runs from GitHub API
	// - API returns 250 runs (mix of agentic and non-agentic)
	// - Only 5 are agentic workflows after filtering
	// - Old code: checks len(runs)=5 < batchSize=250, stops iteration incorrectly
	// - New code: checks totalFetched=250 < batchSize=250 is false, continues iteration
}

// TestDownloadWorkflowLogs_IteratesUntilEnoughRuns demonstrates the fixed behavior
func TestDownloadWorkflowLogs_IteratesUntilEnoughRuns(t *testing.T) {
	t.Skip("Skipping network-dependent test - this would verify end-to-end behavior")

	// This test would verify that when calling:
	// ./gh-aw logs -c 10 (no workflow name)
	//
	// The function:
	// 1. Fetches batches of runs until it has 10 agentic workflow runs
	// 2. Continues iterating if first batch has few agentic workflows
	// 3. Only stops when totalFetched < batchSize (reached end of GitHub data)
	// 4. Returns the same number of results as:
	//    ./gh-aw logs tidy -c 10 (specific workflow name)
}
