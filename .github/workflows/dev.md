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
imports:
  - shared/mcp/tavily.md
---

Search about the latest javascript framework trends.
