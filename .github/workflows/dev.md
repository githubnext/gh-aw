---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - codex*
name: Dev
engine: codex
tools:
  github:
    allowed: [list_pull_requests, get_pull_request]
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
---

List the last 5 merged pull requests (use `github list_pull_requests`) in this repository.