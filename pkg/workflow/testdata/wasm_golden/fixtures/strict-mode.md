---
name: strict-mode-test
description: Workflow with strict mode enabled
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
strict: true
timeout-minutes: 10
---

# Mission

Run a strict mode workflow that enforces additional validation.
