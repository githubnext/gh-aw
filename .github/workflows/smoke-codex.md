---
description: Smoke test workflow that validates Codex engine functionality by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex
engine: codex
tools:
  github:
safe-outputs:
    staged: true
    add-comment:
timeout-minutes: 10
strict: true
---

Review the last 2 merged pull requests in the ${{ github.repository }} repository and add a comment to the current pull request with a summary.