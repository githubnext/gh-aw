---
# DeepWiki MCP Server
# Remote HTTP MCP server for GitHub repository documentation and search
#
# No authentication required - public service
# Documentation: https://mcp.deepwiki.com/
#
# Available tools:
#   - read_wiki_structure: Retrieves documentation topics for a repo
#   - read_wiki_contents: Views documentation about a repo
#   - ask_question: AI-powered Q&A about a repo
#
# Usage:
#   imports:
#     - shared/mcp/deepwiki.md

mcp-servers:
  deepwiki:
    url: "https://mcp.deepwiki.com/sse"
    allowed:
      - read_wiki_structure
      - read_wiki_contents
      - ask_question
---
