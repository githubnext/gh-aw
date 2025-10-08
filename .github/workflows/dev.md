---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
name: Dev
engine: claude
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
imports:
  - shared/gh-aw-mcp.md
---

- Report the status of the github agentic workflows in this repository.

If status fails, give up.

- Summarize the logs information for the last 24h of activity in agentic workflows.
