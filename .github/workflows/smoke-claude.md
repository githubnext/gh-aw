---
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
name: Smoke Claude
engine:
  id: claude
  max-turns: 15
tools:
  github:
    allowed:
      - search_pull_requests
      - pull_request_read
safe-outputs:
    staged: true
    create-issue:
      min: 1
timeout_minutes: 10
strict: true
---

Search for the last 5 merged pull requests in this repository using search filters and post a summary in an issue.