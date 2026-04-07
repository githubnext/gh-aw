---
title: WorkQueueOps
description: Process a queue of work items using GitHub issues, sub-issues, cache-memory, or Discussions as durable queue backends
sidebar:
  badge: { text: 'Queue-based', variant: 'note' }
---

WorkQueueOps is a pattern for systematically processing a backlog of work items with agentic workflows. Instead of processing everything at once, work is queued, tracked, and consumed incrementally — surviving interruptions, rate limits, and multi-day horizons.

## When to Use WorkQueueOps

- **Large backlogs** — hundreds of issues, PRs, or files that cannot all be processed in a single run
- **Idempotent operations** — safe to retry if a run is interrupted (label application, comment posting, code formatting)
- **Progress visibility** — stakeholders need to see what has been done and what remains
- **Resumable work** — the queue should survive workflow cancellations or runner restarts

## Queue Strategy 1: Issue Checklist as Queue

Use GitHub issue checkboxes as a lightweight, human-readable queue. The agent reads the issue body, finds unchecked items, processes each one, and checks it off.

**Best for:** Small-to-medium batches (< 100 items), teams that want visibility into progress without extra tooling.

**Caveats:** Concurrent runs can cause race conditions — use [Concurrency](/gh-aw/reference/concurrency/) controls. Each update requires a safe-output `edit-issue` call, which counts against rate limits.

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      queue_issue:
        description: "Issue number containing the checklist queue"
        required: true

tools:
  github:
    toolsets: [issues]

safe-outputs:
  edit-issue: true
  add-comment:
    max: 1

concurrency:
  group: workqueue-${{ inputs.queue_issue }}
  cancel-in-progress: false
---

# Checklist Queue Processor

You are processing a work queue stored as checkboxes in issue #${{ inputs.queue_issue }}.

## Instructions

1. Read issue #${{ inputs.queue_issue }} and find all unchecked items (`- [ ]`).
2. For each unchecked item (process at most 10 per run to stay within time limits):
   a. Perform the required work described in the item.
   b. Check it off by editing the issue body, changing `- [ ]` to `- [x]` for that item.
3. After processing, add a comment summarizing what was completed and what remains.
4. If all items are checked, close the issue with a summary comment.

Focus on correctness — partial progress is better than skipping items.
```

## Queue Strategy 2: Sub-Issues as Queue

Create one sub-issue per work item. The agent queries open sub-issues of a parent tracking issue, processes each one, and closes it when done.

**Best for:** Work items that benefit from their own discussion thread, review comments, and individual status tracking. Scales to hundreds of items.

**Caveats:** Requires the `parent` feature for sub-issues. Closing issues generates notifications — use `max:` limits on `close-issue` safe outputs to avoid notification storms.

```aw wrap
---
on:
  schedule:
    - cron: "0 * * * *"   # Every hour
  workflow_dispatch:

tools:
  github:
    toolsets: [issues]

safe-outputs:
  add-comment:
    max: 5
  close-issue:
    max: 5

concurrency:
  group: sub-issue-queue
  cancel-in-progress: false
---

# Sub-Issue Queue Processor

You are processing a queue of open sub-issues. The parent tracking issue is labeled `queue-tracking`.

## Instructions

1. Find the open issue labeled `queue-tracking` — this is the queue parent.
2. List its open sub-issues. Process at most 5 per run (to stay within time limits).
3. For each sub-issue:
   a. Read the issue body to understand the required work.
   b. Perform the work described.
   c. Add a comment with the result.
   d. Close the sub-issue when complete.
4. Add a progress comment on the parent tracking issue: how many items remain open.

If no sub-issues are open, post a comment on the parent issue saying the queue is empty and stop.
```

## Queue Strategy 3: Cache-Memory Queue

Store queue state as a JSON file in [cache-memory](/gh-aw/reference/cache-memory/). Each run loads the file, picks up where the last run left off, and saves the updated state.

**Best for:** Large queues, multi-day processing horizons, and scenarios where items are generated programmatically (from API responses, file lists, database exports).

**Caveats:** Cache-memory is scoped to a single branch. Use filesystem-safe timestamps in filenames (e.g., `YYYY-MM-DD-HH-MM-SS-sss` — no colons). Ensure filenames never contain colons or special characters.

```aw wrap
---
on:
  schedule:
    - cron: "0 6 * * 1-5"  # Weekdays at 6 AM
  workflow_dispatch:

tools:
  cache-memory: true
  github:
    toolsets: [repos, issues]
  bash:
    - "jq"

safe-outputs:
  add-comment:
    max: 10
  add-labels:
    allowed: [processed, needs-review]
    max: 10
---

# Cache-Memory Queue Processor

You process items from a persistent JSON queue stored in cache-memory.

## Queue File

The queue lives at `/tmp/gh-aw/cache-memory/workqueue.json` with this structure:

```json
{
  "pending": ["item-1", "item-2"],
  "in_progress": [],
  "completed": ["item-0"],
  "failed": [],
  "last_run": "2026-04-07-06-00-00"
}
```

## Instructions

1. Load the queue file. If it doesn't exist, initialize it by listing all open issues
   without the label `processed` and populating `pending` with their numbers.
2. Move up to 10 items from `pending` to `in_progress`.
3. For each item:
   a. Perform the required operation (add label, post comment, etc.).
   b. On success: move from `in_progress` to `completed`.
   c. On failure: move to `failed` with an error note.
4. Save the updated queue JSON back to `/tmp/gh-aw/cache-memory/workqueue.json`.
5. Report progress: X completed, Y failed, Z remaining.

If `pending` is empty, announce that the queue is exhausted and the job is complete.
```

## Queue Strategy 4: Discussion-Based Queue

Use a GitHub Discussion to track pending work items. Replies to the discussion represent work items; their resolved/unresolved state drives queue membership.

**Best for:** Community-sourced queues, async collaboration, and scenarios where humans need to inspect or modify queue items before or after processing.

**Caveats:** Discussion queries require `discussions` in the GitHub toolset. Marking comments as resolved requires `discussions: write` permission in safe-outputs.

```aw wrap
---
on:
  schedule:
    - cron: "0 8 * * *"   # Daily at 8 AM
  workflow_dispatch:

tools:
  github:
    toolsets: [discussions]

safe-outputs:
  add-comment:
    max: 5
  create-discussion:
    title-prefix: "[queue-log] "
    category: "General"

concurrency:
  group: discussion-queue
  cancel-in-progress: false
---

# Discussion Queue Processor

A GitHub Discussion titled "Work Queue" in category "General" tracks pending items.
Each unresolved top-level reply in that discussion is a work item.

## Instructions

1. Find the discussion titled "Work Queue" in category "General".
2. List all unresolved replies (comments where `isAnswered: false`).
3. For each unresolved reply (process at most 5 per run):
   a. Parse the reply body to extract the work item description.
   b. Perform the described work.
   c. Reply to that comment with the result.
4. Create a summary discussion post (category "General") documenting what was processed today.

Stop after 5 items to avoid rate limits. Remaining items will be processed on the next scheduled run.
```

## Idempotency and Concurrency

All WorkQueueOps patterns should be **idempotent**: running the same item twice should not cause double processing.

| Technique | How |
|-----------|-----|
| Check before acting | Query current state (label present? comment exists?) before making changes |
| Atomic state updates | Write queue state in a single step; avoid partial updates |
| Concurrency groups | Use `concurrency.group` to prevent parallel runs on the same queue |
| Retry budgets | Track failed items separately; set a retry limit before giving up |

```yaml wrap
# Prevent concurrent queue runs
concurrency:
  group: my-queue-processor
  cancel-in-progress: false  # Don't cancel in-progress runs
```

## Related Pages

- [BatchOps](/gh-aw/patterns/batch-ops/) — Process large volumes in parallel chunks rather than sequentially
- [TaskOps](/gh-aw/patterns/task-ops/) — Research → Plan → Assign pattern for developer-supervised work
- [Cache Memory](/gh-aw/reference/cache-memory/) — Persistent state storage across workflow runs
- [Repo Memory](/gh-aw/reference/repo-memory/) — Git-committed persistent state for cross-branch sharing
- [Concurrency](/gh-aw/reference/concurrency/) — Prevent race conditions in queue-based workflows
