---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    command: npx
    args: ["-y", "@microsoft/markitdown"]
---

# Test Copilot Markitdown MCP

This is a test workflow to verify Copilot's integration with Microsoft's markitdown MCP server.

Please use the markitdown MCP server to:
1. Convert a sample HTML file to markdown
2. Convert a PDF document to markdown
3. Show the available conversion capabilities