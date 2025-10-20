# Logs Command Filtering Fix

## Problem Description

The `logs` command was not finding all workflow runs when no workflow name was specified.

**Reproduction:**
```bash
./gh-aw logs tidy -c 10    # Returns 10 runs
./gh-aw logs -c 10          # Returns fewer than 10 runs (inconsistent)
```

## Root Cause

The issue was in the pagination logic in `pkg/cli/logs.go`:

1. `listWorkflowRunsWithPagination()` fetches workflow runs from GitHub API
2. When no workflow name is specified, it filters results to only agentic workflows
3. The iteration loop checked if `len(filteredRuns) < batchSize` to detect end of data
4. This caused premature termination when few agentic workflows were in a batch

**Example scenario:**
- Request 10 agentic workflow runs
- First batch: Fetch 250 runs from API → Only 5 are agentic workflows after filtering
- **Bug**: `len(filteredRuns)=5 < batchSize=250` → Stop iteration ❌
- **Expected**: Continue iterating to find more agentic workflows ✓

## Solution

Modified `listWorkflowRunsWithPagination()` to return two values:
1. Filtered workflow runs (agentic only when no workflow name specified)
2. Total count fetched from GitHub API (before filtering)

Changed the end-of-data check from:
```go
if len(runs) < batchSize {  // WRONG: uses filtered count
    break
}
```

To:
```go
if totalFetched < batchSize {  // CORRECT: uses API response count
    break
}
```

This ensures iteration continues until:
- We have enough agentic workflow runs, OR
- We truly reach the end of GitHub data (API returns fewer than requested)

## Files Changed

1. **pkg/cli/logs.go**
   - Modified `listWorkflowRunsWithPagination()` signature to return `([]WorkflowRun, int, error)`
   - Added `totalFetched` tracking before filtering
   - Updated end-of-data check to use `totalFetched`
   - Added comprehensive comments explaining the fix

2. **pkg/cli/logs_test.go**
   - Updated test to handle new return value from `listWorkflowRunsWithPagination()`

3. **pkg/cli/logs_filtering_test.go** (new)
   - Added documentation tests explaining the expected behavior
   - Tests are skipped (network-dependent) but serve as documentation

## Testing

All existing unit tests pass:
```bash
make test-unit  # ✓ All tests pass
make lint       # ✓ No issues
make build      # ✓ Compiles successfully
```

## Impact

This fix ensures consistent behavior:
- `./gh-aw logs -c 10` now returns 10 agentic workflow runs (not fewer)
- `./gh-aw logs tidy -c 10` behavior unchanged (still returns 10 runs)
- No performance impact (still uses efficient batch fetching)
- No breaking changes (CLI interface unchanged)
