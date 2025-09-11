---
on:
  workflow_dispatch:
  
safe-outputs:
  mcp: true
  create-issue:
    title-prefix: "[MCP Test] "
    labels: [test, mcp, automated]
    max: 1
  add-issue-comment:
    max: 1

engine: claude
---

# MCP Safe Outputs Test

This workflow tests the new MCP-based safe outputs functionality. With `mcp: true` enabled in the safe-outputs configuration, the workflow will use MCP tools instead of writing directly to JSONL files.

Create a test issue with the title "MCP Safe Outputs Test Issue" and body describing that this was created via MCP tools. Then add a comment to the issue confirming that MCP tool integration is working.