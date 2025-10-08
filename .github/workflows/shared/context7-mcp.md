---
mcp-servers:
  context7:
    url: "https://mcp.context7.com/mcp?apiKey=${{ secrets.CONTEXT7_API_KEY }}"
    allowed:
      - get-library-docs
      - resolve-library-id
---

<!--

# Context7 MCP Server
# Vector database and semantic search from Upstash
#
# Provides semantic search capabilities over your data using vector embeddings
# Documentation: https://github.com/upstash/context7
#
# Available tools:
#   - get-library-docs: Get library documentation
#   - resolve-library-id: Resolve library identifiers
#
# Usage:
#   imports:
#     - shared/context7-mcp.md

-->
