---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  set-issue-type:
    allowed: ["Bug", "Feature", "Task"]
    max: 2
---

# Test Copilot Set Issue Type

This workflow tests the set-issue-type safe output type with Copilot engine.

Please set the type of issue #1 to "Bug".
