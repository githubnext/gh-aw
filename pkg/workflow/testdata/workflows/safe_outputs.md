---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
tools:
  github:
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[analysis] "
    labels: [automation]
    max: 1
  add-comment:
    max: 1
timeout-minutes: 10
---

# Safe Outputs Workflow

A workflow that uses safe outputs for creating issues and comments.
