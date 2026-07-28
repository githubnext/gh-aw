---
# Brave Search MCP Server
# SECURITY: docker.io/mcp/brave-search has Critical/High CVEs with no upstream fix available (issue #48546).
# The container definition has been removed until a patched image is published upstream.
# To re-enable, restore the mcp-servers block and update the pinned digest in actions-lock.json.
#
# Requires BRAVE_API_KEY secret
# Get API key from: https://brave.com/search/api/
#
# Available tools (when enabled):
#   - brave_web_search: Search the web using Brave Search
#   - brave_local_search: Search for local businesses and places
#
# Usage:
#   imports:
#     - shared/mcp/brave.md
---
