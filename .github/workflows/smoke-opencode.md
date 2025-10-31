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
name: Smoke OpenCode
imports:
  - shared/opencode.md
tools:
  github:
    toolsets: [pull_requests]
    allowed:
      - list_pull_requests
      - get_pull_request
  safety-prompt: false
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 5
strict: true
---

Review the last 5 merged pull requests in this repository and post summary in an issue.
