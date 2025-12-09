---
on:
  issues:
    types: [opened]
permissions:
  issues: read
engine: copilot
timeout-minutes: 5
safe-outputs:
  add-comment:
---

# Simple Test Workflow

This is a simple test workflow for golden testing.

When an issue is opened, add a comment saying "Hello from simple workflow!".
