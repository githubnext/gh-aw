---
name: Code Simplifier
description: Analyzes recently modified code and creates pull requests with simplifications that improve clarity, consistency, and maintainability while preserving functionality
on:
  schedule: daily
  skip-if-match: 'is:pr is:open in:title "[code-simplifier]"'

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: code-simplifier

imports:
  - shared/reporting.md

safe-outputs:
  create-pull-request:
    title-prefix: "[code-simplifier] "
    labels: [refactoring, code-quality, automation]
    reviewers: [copilot]
    expires: 7d

tools:
  github:
    toolsets: [default]

timeout-minutes: 30
strict: true
---

<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
@./agentics/code-simplifier.md
