---
description: Test workflow for expires field on discussions
on:
  workflow_dispatch:
permissions:
  contents: read
  discussions: write
engine: copilot
safe-outputs:
  create-discussion:
    expires: 7  # Discussion expires in 7 days
    title-prefix: "[Test] "
    labels:
      - test
timeout-minutes: 5
---

# Test Expires Field

This is a test workflow to verify the expires field works correctly for discussions.

The discussion created by this workflow will automatically expire and be closed after 7 days.
