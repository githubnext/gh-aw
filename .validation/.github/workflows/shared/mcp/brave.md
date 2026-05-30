---
# Brave Search MCP Server  
# Container-based MCP server for web search using Brave Search API
#
# Requires BRAVE_API_KEY secret
# Get API key from: https://brave.com/search/api/
#
# Available tools:
#   - brave_web_search: Search the web using Brave Search
#   - brave_local_search: Search for local businesses and places
#
# Usage:
#   imports:
#     - shared/mcp/brave.md

mcp-servers:
  brave-search:
    container: "docker.io/mcp/brave-search"
    env:
      BRAVE_API_KEY: "${{ secrets.BRAVE_API_KEY }}"
    # Security decision (2026-05-19): keep wildcard for this read-only public search server.
    # The server surface is intentionally small and used as a general-purpose search primitive.
    allowed: ["*"]
---
