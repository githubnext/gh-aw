---
name: push-with-head-commit-test
description: Workflow triggered by push events that uses head_commit.id (Git SHA)
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: copilot
timeout-minutes: 10
---

# Mission

When code is pushed to main, analyze the commit ${{ github.event.head_commit.id }} and provide a summary of the changes.
