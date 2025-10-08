---
mcp-servers:
  gh-aw:
    command: "./gh-aw"
    args: ["mcp-server"]
    env:
      GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---

## gh-aw MCP Server

The `gh-aw` MCP server provides access to gh-aw CLI tools through the Model Context Protocol:

- **status** - Show status of agentic workflow files
- **compile** - Compile markdown workflow files to YAML (validation always enabled)
- **logs** - Download and analyze workflow logs (output forced to `/tmp/aw-mcp/logs`)
- **audit** - Investigate a workflow run and generate a report (output forced to `/tmp/aw-mcp/logs`)

The MCP server runs with GITHUB_TOKEN access for GitHub API operations, maintaining security isolation between the workflow process and token access.

### Security

- **Token Isolation**: GITHUB_TOKEN is only accessible to the MCP server subprocess, not the main workflow process
- **Fixed Output Directories**: Logs and audit outputs are written to `/tmp/aw-mcp/logs` to prevent directory traversal
- **Always-Enabled Validation**: The compile tool always validates workflows for security
