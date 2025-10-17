---
on: 
  workflow_dispatch:
  command:
    name: dev
  stop-after: "2025-11-16 00:00:00"
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
    assign-to-bot: copilot
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
---

Write a poem about the last 3 pull requests and publish an issue.
