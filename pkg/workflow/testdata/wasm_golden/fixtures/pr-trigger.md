---
name: pr-trigger-test
description: Workflow triggered by pull request events
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  pull-requests: read
engine: copilot
timeout-minutes: 15
---

# Mission

Review pull requests when opened or updated. Provide constructive feedback.
