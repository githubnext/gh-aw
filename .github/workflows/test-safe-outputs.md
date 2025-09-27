---
name: Test Safe Outputs
on: push
engine: claude
safe-outputs:
  create-issue:
    max: 3
  missing-tool: {}
---

Test safe outputs workflow with MCP server integration.
