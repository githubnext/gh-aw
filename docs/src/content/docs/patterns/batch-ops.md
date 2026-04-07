---
title: BatchOps
description: Process large volumes of work in parallel or chunked batches using matrix jobs, rate-limit-aware throttling, and result aggregation
sidebar:
  badge: { text: 'Batch processing', variant: 'caution' }
---

BatchOps is a pattern for processing large volumes of work items efficiently. Instead of iterating sequentially through hundreds of items in a single workflow run, BatchOps splits work into chunks, parallelizes where possible, handles partial failures gracefully, and aggregates results into a consolidated report.

## When to Use BatchOps vs Sequential Processing

| Scenario | Recommendation |
|----------|----------------|
| < 50 items, order matters | Sequential ([WorkQueueOps](/gh-aw/patterns/workqueue-ops/)) |
| 50–500 items, order doesn't matter | BatchOps with chunked processing |
| > 500 items, high parallelism safe | BatchOps with matrix fan-out |
| Items have dependencies on each other | Sequential (WorkQueueOps) |
| Items are fully independent | BatchOps (any strategy) |
| Strict rate limits or quotas | Rate-limit-aware batching |

## Batch Strategy 1: Chunked Processing

Split work into fixed-size pages using `GITHUB_RUN_NUMBER` or timestamps. Each run processes one page, picking up the next slice automatically on the next scheduled run.

**Best for:** Scheduled maintenance workflows that should run indefinitely, working through a large backlog a page at a time without manual intervention.

**Caveats:** Items are not processed in priority order. Use a stable sort key (creation date, issue number) so pagination is deterministic.

```aw wrap
---
on:
  schedule:
    - cron: "0 2 * * 1-5"  # Weekdays at 2 AM
  workflow_dispatch:

tools:
  github:
    toolsets: [issues]
  bash:
    - "jq"
    - "date"

safe-outputs:
  add-labels:
    allowed: [stale, needs-triage, archived]
    max: 30
  add-comment:
    max: 30

steps:
  - name: compute-page
    run: |
      PAGE_SIZE=25
      # Use run number mod to cycle through pages; reset every 1000 runs
      PAGE=$(( (GITHUB_RUN_NUMBER % 1000) * PAGE_SIZE ))
      echo "page_offset=$PAGE" >> "$GITHUB_OUTPUT"
      echo "page_size=$PAGE_SIZE" >> "$GITHUB_OUTPUT"
---

# Chunked Issue Processor

You are processing a chunk of issues. This run covers offset ${{ steps.compute-page.outputs.page_offset }}
with page size ${{ steps.compute-page.outputs.page_size }}.

## Instructions

1. List issues sorted by creation date (oldest first), skipping the first
   ${{ steps.compute-page.outputs.page_offset }} issues, and taking
   ${{ steps.compute-page.outputs.page_size }} issues.
2. For each issue in the chunk:
   - Add label `stale` if last updated more than 90 days ago with no recent comments
   - Add label `needs-triage` if it has no labels at all
   - Post a comment if the issue is stale, letting the author know it will be closed in 14 days
3. Summarize: how many issues were labeled, how many comments posted, any errors.
```

## Batch Strategy 2: Fan-Out with Matrix

Use GitHub Actions matrix to run multiple batch workers in parallel, each responsible for a non-overlapping shard of the work.

**Best for:** Time-sensitive bulk operations where you want all items processed in a single workflow run. Requires clear shard assignment so workers don't overlap.

**Caveats:** Matrix jobs count against your GitHub Actions concurrency limits. Each shard runs as a separate job with its own token and API rate limit quota. Use shard index modulo to partition items without a coordination database.

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      total_shards:
        description: "Number of parallel workers"
        default: "4"
        required: false

jobs:
  batch:
    strategy:
      matrix:
        shard: [0, 1, 2, 3]
      fail-fast: false   # Continue other shards even if one fails

tools:
  github:
    toolsets: [issues, pull_requests]

safe-outputs:
  add-labels:
    allowed: [reviewed, duplicate, wontfix]
    max: 50
---

# Matrix Batch Worker — Shard ${{ matrix.shard }} of ${{ inputs.total_shards }}

You are shard number ${{ matrix.shard }} in a ${{ inputs.total_shards }}-shard batch job.

## Shard Assignment

Process only issues where `(issue_number % ${{ inputs.total_shards }}) == ${{ matrix.shard }}`.
This ensures no two shards process the same issue.

## Instructions

1. List all open issues (up to 500). Keep only those where `issue_number mod ${{ inputs.total_shards }} == ${{ matrix.shard }}`.
2. For each assigned issue:
   - Check if it is a duplicate (search for similar titles or content).
   - Add label `reviewed` to indicate it has been processed in this batch.
   - If a clear duplicate is found, add label `duplicate` and reference the original.
3. Report: total issues in your shard, how many labeled, any failures.
```

## Batch Strategy 3: Rate-Limit-Aware Batching

Throttle API calls by processing items in small sub-batches with explicit pauses between them.

**Best for:** Workflows that call external APIs with strict quotas (e.g., a third-party review service), or when you want to avoid consuming your entire GitHub API quota in one run.

**Caveats:** Slower than unbounded processing but dramatically reduces rate-limit errors. Use [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) for built-in throttling.

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      batch_size:
        description: "Items per sub-batch"
        default: "10"
      pause_seconds:
        description: "Seconds to pause between sub-batches"
        default: "30"

tools:
  github:
    toolsets: [repos, issues]
  bash:
    - "sleep"
    - "jq"

safe-outputs:
  add-comment:
    max: 100
  add-labels:
    allowed: [labeled-by-bot]
    max: 100
---

# Rate-Limited Batch Processor

Process all open issues across the repository while respecting API rate limits.

## Instructions

Process issues in sub-batches of ${{ inputs.batch_size }} at a time:

1. Fetch all open issue numbers (use pagination if needed).
2. Split into sub-batches of size ${{ inputs.batch_size }}.
3. For each sub-batch:
   a. Process each issue: read its body, determine the correct label, add the label.
   b. After processing the entire sub-batch, pause for ${{ inputs.pause_seconds }} seconds
      before starting the next sub-batch. This avoids secondary rate-limit errors.
4. If you encounter a rate-limit error (HTTP 429), pause for 60 seconds and retry
   the current item once before marking it as failed.
5. Report a final summary: total processed, total failed, total skipped.

Keep a count of API calls and pause proactively when approaching limits.
```

## Batch Strategy 4: Result Aggregation

Collect results from multiple batch workers or runs and aggregate them into a single summary issue or discussion.

**Best for:** Long-running batch campaigns where stakeholders need consolidated reporting, audit trails, or comparison across multiple runs.

**Caveats:** Aggregation requires a stable identifier (issue number, discussion number) to append results to. Use [cache-memory](/gh-aw/reference/cache-memory/) to store intermediate results if runs span multiple days.

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      report_issue:
        description: "Issue number to aggregate results into"
        required: true

tools:
  cache-memory: true
  github:
    toolsets: [issues, repos]
  bash:
    - "jq"

safe-outputs:
  add-comment:
    max: 1
  edit-issue: true

steps:
  - name: collect-results
    run: |
      # Aggregate results from all result files written by previous batch runs
      RESULTS_DIR="/tmp/gh-aw/cache-memory/batch-results"
      if [ -d "$RESULTS_DIR" ]; then
        jq -s '
          {
            total_processed: (map(.processed) | add // 0),
            total_failed: (map(.failed) | add // 0),
            total_skipped: (map(.skipped) | add // 0),
            runs: length,
            errors: (map(.errors // []) | add // [])
          }
        ' "$RESULTS_DIR"/*.json > /tmp/gh-aw/cache-memory/aggregate.json
        cat /tmp/gh-aw/cache-memory/aggregate.json
      else
        echo '{"total_processed":0,"total_failed":0,"total_skipped":0,"runs":0,"errors":[]}' \
          > /tmp/gh-aw/cache-memory/aggregate.json
      fi
---

# Batch Result Aggregator

You are aggregating results from all batch processing runs into issue #${{ inputs.report_issue }}.

## Aggregate Data

The aggregated statistics are at `/tmp/gh-aw/cache-memory/aggregate.json`.
Individual run result files are in `/tmp/gh-aw/cache-memory/batch-results/`.

## Instructions

1. Read `/tmp/gh-aw/cache-memory/aggregate.json` for totals.
2. Read each individual result file to understand per-run breakdowns.
3. Update issue #${{ inputs.report_issue }} body with a Markdown table:
   - Summary row: total processed / failed / skipped across all runs
   - Per-run breakdown table
   - List of any errors or items requiring manual intervention
4. Add a comment to issue #${{ inputs.report_issue }} with the final status:
   "Batch complete ✅" if no failures, or "Batch complete with failures ⚠️" and a list of failed items.
5. If there are failed items, create sub-issues for each so they can be retried.
```

## Error Handling and Partial Failures

Batch workflows must be resilient to individual item failures.

### Retry Pattern

```aw wrap
---
tools:
  cache-memory: true
  bash:
    - "jq"
---

# Retry Failed Items

Load `/tmp/gh-aw/cache-memory/workqueue.json`. Focus only on items in `failed`
where `retry_count < 3`. For each:
1. Attempt to process the item again.
2. On success: move to `completed`, remove from `failed`.
3. On second failure: increment `retry_count`. Leave in `failed`.
4. On third failure: move to `permanently_failed` for human review.
5. Save the updated queue.
```

### Failure Isolation

- Use `fail-fast: false` in matrix jobs so one shard failure doesn't cancel others
- Write per-item results before moving to the next item
- Store errors with enough context to diagnose and retry without re-reading everything

## Real-World Example: Updating Labels Across 100+ Issues

This example processes a label migration (rename `bug` to `type:bug`) across all open and closed issues.

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      dry_run:
        description: "Preview changes without applying them"
        default: "true"

tools:
  github:
    toolsets: [issues]
  bash:
    - "jq"

safe-outputs:
  add-labels:
    allowed: [type:bug]
    max: 200
  remove-labels:
    allowed: [bug]
    max: 200
  add-comment:
    max: 1

concurrency:
  group: label-migration
  cancel-in-progress: false
---

# Label Migration: `bug` → `type:bug`

Migrate all issues with the label `bug` to use `type:bug` instead.

## Instructions

1. List all issues (open and closed) that have the label `bug`. Use pagination to
   retrieve all of them — there may be more than 100.
2. If `${{ inputs.dry_run }}` is `true`:
   - Report how many issues would be updated but make no changes.
   - Add a comment summarizing the planned migration.
3. If `${{ inputs.dry_run }}` is `false`:
   - For each issue: add label `type:bug`, then remove label `bug`.
   - Process in sub-batches of 20 issues, pausing 15 seconds between batches.
   - Track successes and failures.
4. Add a final comment with the migration summary: total issues updated, any failures,
   and a link to search for remaining `bug` labels to verify completeness.
```

## Related Pages

- [WorkQueueOps](/gh-aw/patterns/workqueue-ops/) — Sequential queue processing with issue checklists, sub-issues, cache-memory, and Discussions
- [TaskOps](/gh-aw/patterns/task-ops/) — Research → Plan → Assign for developer-supervised work
- [Cache Memory](/gh-aw/reference/cache-memory/) — Persistent state storage across workflow runs
- [Repo Memory](/gh-aw/reference/repo-memory/) — Git-committed persistent state
- [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) — Built-in throttling for API-heavy workflows
- [Concurrency](/gh-aw/reference/concurrency/) — Prevent overlapping batch runs
