---
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
  contents: read
  actions: read
tools:
  github:
    toolsets: [default]
timeout-minutes: 10
---

# Dev

## Current Context
- **Repository**: ${{ github.repository }}

## Rule Definition

Analyze the codebase in repository for compliance.