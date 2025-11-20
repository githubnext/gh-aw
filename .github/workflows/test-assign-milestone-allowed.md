---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  assign-milestone:
    allowed: [v1.0, v2.0, Sprint 1]
    max: 3
---

# Test Assign Milestone with Allowed List

This workflow demonstrates the `allowed` field for `assign-milestone`.

With an allowed list of milestones, the workflow will only assign issues to
milestones that match the configured names.

Please assign:
- Issue #1 to milestone "v1.0"
- Issue #2 to milestone "v2.0"
- Issue #3 to milestone "Sprint 1"
