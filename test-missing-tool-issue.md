---
name: Test Missing Tool Issue Creation
engine: copilot
on:
  workflow_dispatch:
safe-outputs:
  missing-tool:
    create-issue: true
    title-prefix: "[test missing tool]"
    labels:
      - bug
      - missing-tool-test
    max: 5
---

# Test Missing Tool Issue Creation

This is a test workflow to verify that the missing-tool safe output can create issues.

For testing purposes, this workflow intentionally uses tools that don't exist to trigger the missing-tool reporting.
