---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
network: defaults
tools:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    command: npx
    args: ["-y", "@microsoft/markitdown"]
---

# Test Claude Markitdown MCP

This is a test workflow to verify Claude's integration with Microsoft's markitdown MCP server.

Please use the markitdown MCP server to:
1. Convert a sample HTML file to markdown
2. Convert a PDF document to markdown
3. Show the available conversion capabilities