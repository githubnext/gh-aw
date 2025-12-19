---
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
  contents: read
  actions: read
tools:
  bash:
    - "*"
  github:
    toolsets: [default]
timeout-minutes: 10
---

# Dev

## Rule Definition

Analyze the codebase in repository ${{ github.repository }} for compliance.