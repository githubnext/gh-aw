---
name: push-trigger-test
description: Workflow triggered by push events
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: copilot
timeout-minutes: 10
---

# Mission

When code is pushed to main, review the changes and provide a summary.
