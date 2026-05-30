---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  assign-milestone:
    max: 2
---

# Test Copilot Assign Milestone

This workflow tests the assign-milestone safe output type with Copilot engine.

Please assign issue #1 to milestone #5 and issue #2 to milestone #5.
