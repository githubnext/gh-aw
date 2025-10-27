---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  close-issue:
    required-labels:
      - stale
    outcome:
      - completed
      - not_planned
    max: 3
---

# Test Close Issue Safe Output

This workflow tests the new close-issue safe output type.

When invoked, close issues that have the "stale" label using the close-issue tool from the safeoutputs MCP server.

For testing, you can:
1. Close issue #1 with outcome "completed"
2. Close issue #2 with outcome "not_planned"
