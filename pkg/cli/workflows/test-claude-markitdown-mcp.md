---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
network: defaults
mcp-servers:
  markitdown:
    command: npx
    args: ["-y", "markitdown-mcp-npx"]
---

# Test Claude Markitdown MCP

This is a test workflow to verify Claude's integration with Microsoft's markitdown MCP server.

Please use the markitdown MCP server to:
1. Convert a sample HTML file to markdown
2. Convert a PDF document to markdown
3. Show the available conversion capabilities