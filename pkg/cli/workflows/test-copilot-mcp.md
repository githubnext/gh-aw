---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
tools:
  mcp: "npx -y @modelcontextprotocol/server-filesystem /tmp"
---

# Test Copilot MCP

This is a test workflow to verify Copilot's MCP (Model Context Protocol) capabilities.

Please use the MCP filesystem server to:
1. Create a test file at /tmp/gh-aw/test-mcp.txt
2. Write some sample content to it
3. Read the content back to verify it was written correctly