---
on:
  issues:
    types: [opened]
permissions:
  issues: write
  contents: read
engine: copilot
tools:
  github:
    toolsets: [default]
timeout-minutes: 10
---

# Issue Triage Workflow

Analyze the newly opened issue and add appropriate labels.
