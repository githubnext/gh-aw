---
on: 
  workflow_dispatch:
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
safe-outputs:
  staged: true
  create-issue:
---

Write a poem about the last 3 pull requests and publish an issue.
