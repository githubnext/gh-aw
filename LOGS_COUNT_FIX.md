# Fix: Logs Command Count Parameter Confusion

## Problem

The `listWorkflowRunsWithPagination` function had a confusing parameter name that led to misunderstanding of the algorithm's behavior.

### Confusion

The function parameter was named `count`, which suggested it represented the total number of matching runs the user wants to find (from the `-c` flag). However, the parameter was actually used as the **batch size** for the GitHub CLI API call.

This created confusion between two distinct concepts:
1. **User's count** (`-c` flag): Total number of matching workflow runs to find
2. **API batch size**: Number of runs to fetch per `gh run list` call

### Original Code

```go
func listWorkflowRunsWithPagination(workflowName string, count int, ...) {
    // ...
    if count > 0 {
        args = append(args, "--limit", strconv.Itoa(count))
    }
    // ...
}
```

When called:
```go
runs, totalFetched, err := listWorkflowRunsWithPagination(workflowName, batchSize, ...)
```

The `batchSize` (100 or 250) was passed as `count`, making it unclear whether it represented:
- The user's desired total matching runs, or
- The API batch size per request

## Solution

Renamed the parameter from `count` to `limit` to clarify its purpose:

```go
func listWorkflowRunsWithPagination(workflowName string, limit int, ...) {
    // ...
    if limit > 0 {
        args = append(args, "--limit", strconv.Itoa(limit))
    }
    // ...
}
```

Now it's clear that `limit` represents the API batch size (passed to `gh run list --limit`).

## Algorithm Flow

When a user runs `./gh-aw logs -c 10`:

1. **Initialization**: `count = 10` (user's desired total matching runs)
2. **Loop iteration**:
   - Calculate `batchSize` (100 or 250 depending on conditions)
   - Call `listWorkflowRunsWithPagination(workflowName, batchSize, ...)` 
   - This passes `batchSize` as the `limit` parameter to GitHub CLI: `gh run list --limit 100`
   - Download up to `maxDownloads = count - len(processedRuns)` runs from the batch
   - Continue until `len(processedRuns) >= count` or exhaust available runs

3. **Result**: User gets up to 10 matching workflow runs

## Key Clarifications

- **User's `-c` flag**: Controls the total number of matching runs to return (default: 100)
- **`limit` parameter**: Controls the batch size for each API call (100 or 250)
- **Loop continues**: Until we have enough matching runs OR reach end of available data

## Testing

Added test `TestListWorkflowRunsWithPagination_LimitParameter` to document the parameter semantics.

All unit tests pass successfully after the change.
