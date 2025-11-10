---
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  issues: read
  pull-requests: read
  
name: Smoke Claude
engine:
  id: claude
  max-turns: 15
imports:
  - shared/mcp-pagination.md
tools:
  github:
    toolsets: [repos, pull_requests]
safe-outputs:
    staged: true
    create-issue:
timeout-minutes: 10
strict: true
---

Review the last 2 merged pull requests in this repository and post summary in an issue.