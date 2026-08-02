---
# Notion MCP Server
# SECURITY: docker.io/mcp/notion has Critical/High CVEs with no upstream fix available (issue #49517).
# The container definition has been removed until a patched image is published upstream.
# To re-enable, restore the mcp-servers and safe-outputs blocks and update the pinned digest in actions-lock.json.
#
# Requires NOTION_API_TOKEN secret
#
# Available tools (when enabled):
#   - search_pages: Search for Notion pages
#   - get_page: Get details of a specific page
#   - get_database: Get database schema
#   - query_database: Query database content
---
