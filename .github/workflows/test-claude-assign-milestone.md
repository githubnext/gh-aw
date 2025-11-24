---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  assign-milestone:
    max: 2
strict: false
---

# Test Claude Assign Milestone

This workflow tests the assign-milestone safe output type with Claude engine.

Please assign issue #1 to milestone #5 and issue #2 to milestone #5.
