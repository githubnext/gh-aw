---
description: Test workflow for hide-older-comments field on add-comment safe output
on:
  workflow_dispatch:
  issue_comment:
    types: [created]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tracker-id: test-hide-comments-workflow
safe-outputs:
  add-comment:
    hide-older-comments: true
timeout-minutes: 5
---

# Test Hide Older Comments

This is a test workflow to verify the hide-older-comments field works correctly.

When this workflow runs multiple times on the same issue, it will hide all previous comments from this workflow (identified by the tracker-id) before adding a new comment.

The comment will include a timestamp to help verify the hiding behavior.

Current timestamp: {{ new Date().toISOString() }}
