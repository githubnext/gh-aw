---
on: 
  workflow_dispatch:
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
    assignees: copilot
imports:
  - shared/mcp/tavily.md
---

Write a creative poem about GitHub Agentic Workflows and create an issue with the poem. Assign the issue to copilot.
