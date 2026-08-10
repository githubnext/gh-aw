---
mcp-servers:
  context7:
    # Security decision (2026-08-10): use the official hosted HTTP endpoint instead of the
    # `mcp/context7` container image, which carries Critical/High CVEs (see #51715).
    type: http
    url: "https://mcp.context7.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.CONTEXT7_API_KEY }}"
    allowed:
      - query-docs
      - resolve-library-id
---

<!--

# Context7 MCP Server
# Up-to-date code documentation for any library from Upstash
#
# Fetches version-specific documentation and code examples for libraries and frameworks.
# Helps generate accurate, up-to-date code without hallucinated APIs or outdated examples.
# Uses the hosted remote HTTP MCP server, authenticated with CONTEXT7_API_KEY.
# Documentation: https://github.com/upstash/context7
#
# Available tools:
#   - resolve-library-id: Resolves a library name into a Context7-compatible library ID
#   - query-docs: Retrieves documentation for a library using a Context7-compatible library ID
#
# Usage:
#   imports:
#     - shared/mcp/context7.md
#
# Example prompt:
#   "Create Next.js middleware that checks for JWT. use context7"
#   "Implement authentication with Supabase. use library /supabase/supabase for API and docs."

-->
