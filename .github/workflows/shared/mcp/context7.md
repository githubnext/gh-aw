---
mcp-servers:
  context7:
    container: "mcp/context7"
    env:
      CONTEXT7_API_KEY: "${{ secrets.CONTEXT7_API_KEY }}"
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
#     - shared/mcp/context7.md

-->
