---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
imports:
  - ./shared/otlp-github-app-import.md
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# OTLP GitHub App token minting from import

This workflow verifies `observability.otlp.github-app` is honored when configured in an imported workflow.
