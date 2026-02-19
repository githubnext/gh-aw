---
name: with-safe-outputs-test
description: Workflow with safe output configuration
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
timeout-minutes: 10
tools:
  github:
    toolsets: [issues]
safe-outputs:
  create-issue:
    max: 1
  add-comment:
    target: "*"
---

# Mission

Analyze the repository and create an issue with a summary of findings. Add a comment to existing issues if relevant.
