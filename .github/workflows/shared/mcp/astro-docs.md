---
# Astro Documentation MCP Server
# Remote HTTP MCP server for Astro documentation search and content access
#
# No authentication required - public service
# Documentation: https://docs.astro.build/
#
# Available tools:
#   - Search and query Astro documentation
#   - Access Astro guides, references, and tutorials
#   - Get information about Astro features and APIs
#
# Usage:
#   imports:
#     - shared/mcp/astro-docs.md

mcp-servers:
  astro-docs:
    url: "https://mcp.docs.astro.build/mcp"
    allowed: ["*"]
---
