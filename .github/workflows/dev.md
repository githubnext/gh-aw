---
on: 
  workflow_dispatch:
  command:
    name: dev
name: Dev
engine: claude
permissions:
  contents: read
  actions: read
imports:
  - shared/mcp/gh-aw.md
safe-outputs:
  threat-detection: false
  create-issue:
    assign-to-bot: copilot
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
---

Write a poem in 3 emojis about the last pull request and publish an issue.
