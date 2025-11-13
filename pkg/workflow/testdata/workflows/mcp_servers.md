---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
engine: copilot
mcp-servers:
  custom-tool:
    container: "example/custom-mcp:latest"
    allowed:
      - custom_function
tools:
  github:
    toolsets: [default]
timeout-minutes: 10
---

# MCP Server Workflow

A workflow that uses custom MCP servers for additional tools.
