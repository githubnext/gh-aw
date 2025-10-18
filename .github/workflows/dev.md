---
on: 
  workflow_dispatch:
  push:
    paths:
      - '.github/workflows/dev.md'
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
safe-outputs:
  threat-detection: false
  create-issue:
    assign-to-bot: copilot
    github-token: ${{ secrets.GH_AW_GITHUB_TOKEN }}
---

Create an issue with a 3 emoji poem.
