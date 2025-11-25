---
mcp-servers:
  linear:
    type: http
    url: "https://mcp.linear.app/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.LINEAR_API_KEY }}"
    allowed:
      - "*"
---

<!--
Linear MCP Server
Project and issue tracking integration via Model Context Protocol

Documentation: https://linear.app/docs/mcp

This shared configuration provides Linear MCP server integration for issue tracking,
project management, and team collaboration via HTTP API with key-based authentication.

Setup:
  1. Generate a Linear API Key:
     - Log in to your Linear workspace
     - Go to Settings > My Account > API
     - Create a Personal API Key
     - Note: The key must have access to the workspaces and data you want to query/modify

  2. Add Repository Secret:
     - LINEAR_API_KEY: Your Linear Personal API Key (required)

  3. Include in Your Workflow:
     imports:
       - shared/mcp/linear.md

Available Tools:
  The Linear MCP server provides tools for:
  - Finding, creating, and updating issues
  - Managing projects and project updates
  - Working with comments and attachments
  - Querying team and user information

  Note: Use `gh aw mcp inspect <workflow> --server linear` to discover available tools.

Connection Type:
  This configuration uses HTTP (Streamable HTTP) transport, which is recommended
  for increased reliability. The server also supports SSE at https://mcp.linear.app/sse

Authentication:
  Uses key-based authentication via Bearer token in the Authorization header.
  This method allows passing API keys directly without the interactive OAuth flow.

Example Usage:
  Search for open issues in the current sprint and summarize their status.

Troubleshooting:
  401 Unauthorized Errors - Verify that:
  - Your LINEAR_API_KEY secret is correctly configured
  - The API key has not expired or been revoked
  - The key has necessary permissions for the requested resources

  500 Internal Server Error:
  - Try clearing saved auth info: rm -rf ~/.mcp-auth
  - Ensure you're using a compatible node version

References:
  - Linear MCP Documentation: https://linear.app/docs/mcp
  - Key Authentication: https://linear.app/docs/mcp#collapsible-28a2f832a8df
  - Linear API Reference: https://developers.linear.app/docs/graphql/working-with-the-graphql-api
-->
