---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
tools:
  github:
    mode: remote
    allowed: [get_repository, list_issues, get_issue]
---

# Test Claude with GitHub Remote MCP

This is a test workflow to verify Claude's ability to use the hosted GitHub MCP server in remote mode.

Please use the remote GitHub MCP server to:
1. Get information about this repository (githubnext/gh-aw)
2. List the first 3 open issues
3. Get details for issue #1 if it exists

The workflow uses `mode: remote` to connect to the hosted GitHub MCP server at https://api.githubcopilot.com/mcp/ with GITHUB_MCP_TOKEN for authentication.
