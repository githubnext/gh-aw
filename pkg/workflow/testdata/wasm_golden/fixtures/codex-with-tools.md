---
name: codex-with-tools-test
description: Codex workflow with tool configurations
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: codex
timeout-minutes: 10
network:
  allowed:
    - defaults
tools:
  bash: true
  github:
    toolsets: [issues, repos]
---

# Mission

Use bash and GitHub tools to analyze the repository and report findings.
