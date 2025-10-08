---
# Context7 MCP Server
# Vector database and semantic search from Upstash
#
# Provides semantic search capabilities over your data using vector embeddings
# Documentation: https://github.com/upstash/context7
#
# Available tools:
#   - search: Semantic search over stored data
#   - insert: Add data to the vector database
#   - query: Query the vector database
#
# Usage:
#   imports:
#     - shared/context7-mcp.md

mcp-servers:
  context7:
    container: "mcp/upstash/context7"
    env:
      UPSTASH_VECTOR_REST_URL: "${{ secrets.UPSTASH_VECTOR_REST_URL }}"
      UPSTASH_VECTOR_REST_TOKEN: "${{ secrets.UPSTASH_VECTOR_REST_TOKEN }}"
    allowed:
      - search
      - query
---
