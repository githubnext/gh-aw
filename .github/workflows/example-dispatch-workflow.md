---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  dispatch-workflow:
    allowed-workflows:
      - audit-workflows.md
      - daily-news.md
---

# Workflow Dispatcher Example

This workflow demonstrates the dispatch-workflow safe output type.

When triggered manually, you can use this to dispatch other workflows in the repository.

Available workflows to dispatch:
- `audit-workflows.md` - Analyzes workflow runs and generates reports
- `daily-news.md` - Generates daily news summaries

To dispatch a workflow, use the `dispatch_workflow` tool from the safe-outputs MCP server.

Example task: Dispatch the `audit-workflows.md` workflow to analyze recent workflow runs.
