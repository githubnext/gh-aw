---
engine: claude
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  my-tool:
    mcp-ref: "vscode"
    allowed: [list_files, read_file]
---

# Weekly file analysis using VSCode MCP configuration

This workflow uses the MCP server configuration defined in `.vscode/mcp.json` to analyze files in the repository.

The tool configuration references the "my-tool" server from VSCode MCP settings.