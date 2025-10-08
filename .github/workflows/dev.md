---
name: Dev
on: 
  workflow_dispatch:
    inputs:
      funny:
        description: 'Make the poem funny'
        required: false
        type: boolean
  push:
    branches:
      - copilot*
      - detection
      - codex*
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