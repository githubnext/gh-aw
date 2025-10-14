---
steps:
  - name: Install Markitdown MCP
    run: pip install markitdown-mcp
mcp-servers:
  markitdown:
    command: "markitdown-mcp"
    allowed: ["*"]
---
