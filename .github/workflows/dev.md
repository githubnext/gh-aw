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
  - shared/mcp/drain3.md
  - shared/mcp/gh-aw.md
safe-outputs:
  create-issue:
tools:
  github:
---

Write a poem in 3 emojis about the last pull request and publish an issue.
