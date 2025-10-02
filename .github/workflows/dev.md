---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
  stop-after: "2026-01-01 00:00:00"
engine: copilot
tools:
  github:
    allowed:
      - list_pull_requests
      - get_pull_request
  web-fetch:
safe-outputs:
    staged: true
    create-issue:
---
Write a poem and post it as an issue.
