---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - parse*
name: Dev
engine: codex
tools:
  github:
    mode: remote
    github-token: "${{ secrets.COPILOT_CLI_TOKEN }}"
    allowed: [list_pull_requests, get_pull_request]
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
---

List the last 5 merged pull requests in this repository.