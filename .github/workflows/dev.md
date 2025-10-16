---
on: 
  command:
    name: dev
  stop-after: "2025-11-16 00:00:00"
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
