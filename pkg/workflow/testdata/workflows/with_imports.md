---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: copilot
tools:
  github:
    toolsets: [default]
imports:
  - shared/common-tools.md
timeout-minutes: 10
---

# Workflow with Imports

A workflow that imports shared configuration.
