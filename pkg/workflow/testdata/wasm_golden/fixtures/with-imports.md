---
name: with-imports-test
description: Workflow with shared component imports
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
timeout-minutes: 10
imports:
  - shared/tools.md
---

# Mission

Use the imported tools to analyze the repository.
