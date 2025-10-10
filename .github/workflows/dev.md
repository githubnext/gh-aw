---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - discussion*
name: Dev
engine: codex
tools:
  github:
    allowed: [list_pull_requests, get_pull_request]
safe-outputs:
    staged: true
    create-discussion:
---

List the last 5 merged pull requests (use `github list_pull_requests`) in this repository
and open a discussion with the list.