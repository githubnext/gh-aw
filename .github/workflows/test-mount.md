---
name: Test Mount
on: issue_comment
engine: copilot
permissions:
  issues: read
  pull-requests: read
network:
  allowed:
    - "github.com"
safe-outputs:
  create-issue: {}
---

# Test

This is a test workflow to verify mounts.
