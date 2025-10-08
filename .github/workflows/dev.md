---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - serena*
name: Dev
engine: copilot
tools:
  github:
    mode: remote
    github-token: "${{ secrets.COPILOT_CLI_TOKEN }}"
    allowed: [list_pull_requests, get_pull_request]
  cache-memory: true
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
imports:
  - shared/serena-mcp.md
---

List the last 5 merged pull requests in this repository.