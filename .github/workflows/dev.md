---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - serena*
name: Dev
engine: codex
tools:
  cache-memory: true
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
imports:
  - shared/serena-mcp.md
---

Use serena to count methods in go sources.