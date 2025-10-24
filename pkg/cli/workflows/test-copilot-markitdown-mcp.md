---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
mcp-servers:
  markitdown:
    command: npx
    args: ["-y", "markitdown-mcp-npx"]
---

# Test Copilot Markitdown MCP

This is a test workflow to verify Copilot's integration with Microsoft's markitdown MCP server.

Please use the markitdown MCP server to:
1. Convert a sample HTML file to markdown
2. Convert a PDF document to markdown
3. Show the available conversion capabilities