---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: codex
safe-outputs:
  assign-milestone:
    max: 2
---

# Test Codex Assign Milestone

This workflow tests the assign-milestone safe output type with Codex engine.

Please assign issue #1 to milestone #5 and issue #2 to milestone #5.
