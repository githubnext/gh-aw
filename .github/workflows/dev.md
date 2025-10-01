---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
engine: copilot
tools:
  github:
    allowed:
      - list_pull_requests
      - get_pull_request
safe-outputs:
    staged: true
    create-issue:
---
Write a poem and post it as an issue.
