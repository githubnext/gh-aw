---
description: Automatically closes discussions created by github-actions bot that are older than 1 week
on:
  workflow_dispatch:
  schedule:
    - cron: "0 6 * * 0"  # Weekly on Sundays at 6 AM UTC
permissions:
  contents: read
  actions: read
  discussions: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [default, discussions]
safe-outputs:
  close-discussion:
    max: 10
timeout-minutes: 10
strict: true
---

# Close Old Discussions Created by GitHub Actions Bot

This workflow automatically closes discussions that were created by the `github-actions[bot]` user and are older than 1 week.

## Task Requirements

1. **List all open discussions** in the repository `${{ github.repository }}`
2. **Filter discussions** to find those that:
   - Were created by the user `github-actions[bot]`
   - Were created more than 7 days ago (1 week threshold)
3. **Close each matching discussion** with:
   - A comment explaining the closure: "This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago."
   - Resolution reason: "OUTDATED"

## Implementation Instructions

### Step 1: List Open Discussions

Use the GitHub API to list all open discussions in the repository. You can use the `github` tool with the discussions toolset.

### Step 2: Filter Discussions

For each discussion:
- Check if the author login is exactly `github-actions[bot]`
- Calculate the age by comparing the discussion's `createdAt` date with the current date
- Only include discussions that are more than 7 days old (created before 1 week ago)

### Step 3: Close Matching Discussions

For each discussion that matches the criteria, output a `close_discussion` item with:
- **discussion_number**: The discussion number to close
- **body**: "This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago."
- **reason**: "OUTDATED"

Output format (JSONL):
```jsonl
{"type":"close_discussion","discussion_number":123,"body":"This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago.","reason":"OUTDATED"}
```

## Important Notes

- **Maximum closures**: The workflow is configured to close up to 10 discussions per run
- **User filter**: Only close discussions created by `github-actions[bot]`, not by human users
- **Age threshold**: Only close discussions older than 7 days (1 week)
- **Be selective**: If there are no matching discussions, report that no action was needed
- **Safe outputs**: Use the `close_discussion` safe output type which will be processed by the automated system

## Success Criteria

A successful run will:
- ✅ List all open discussions from the repository
- ✅ Correctly identify discussions created by `github-actions[bot]`
- ✅ Accurately calculate discussion age (older than 1 week)
- ✅ Generate appropriate `close_discussion` outputs for matching discussions
- ✅ Respect the maximum of 10 closures per run
- ✅ Provide clear logging of actions taken

Begin the task now. List discussions, filter by author and age, and generate close_discussion outputs for matching items.
