---
on: 
  workflow_dispatch:
  command:
    name: dev
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
    assign-to-bot: copilot
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
    assign-to-bot-github-token: ${{ secrets.GH_AW_GITHUB_ASSIGN_COPILOT_TOKEN }}
---

Write a poem in 3 emojis about the last pull request and publish an issue.
