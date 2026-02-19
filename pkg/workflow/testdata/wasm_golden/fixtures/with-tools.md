---
name: with-tools-test
description: Workflow with various tool configurations
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
timeout-minutes: 10
tools:
  bash: true
  edit:
  github:
    toolsets: [issues, repos]
---

# Mission

Use the available tools to analyze the repository and report findings.
