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
tools:
  edit:
  github:
safe-outputs:
  create-issue:
imports:
  - shared/mcp/tavily.md
---

Search the latest trends about javascript frameworks using tavily tools, and the last 3 pull requests using github tools and print a summary.
