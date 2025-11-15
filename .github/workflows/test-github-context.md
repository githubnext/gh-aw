---
name: Test GitHub Context Share
description: Example workflow demonstrating the github-context.md shared import
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
  issue_comment:
    types: [created]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - shared/github-context.md
safe-outputs:
  add-comment:
    max: 1
timeout-minutes: 5
---

# Test GitHub Context Share

You are testing the GitHub context share import functionality.

Review the GitHub Invocation Context provided above (from the imported shared/github-context.md file).

Create a comment that summarizes the current GitHub context in a friendly way, mentioning:
- The repository and workflow being executed
- The specific event that triggered this workflow (issue, PR, or comment)
- The relevant IDs from the context (issue number, PR number, comment ID, etc.)

Keep the response brief and friendly, showing only the context fields that are actually populated for this event type.
