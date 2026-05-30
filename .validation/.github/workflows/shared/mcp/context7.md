---
mcp-servers:
  context7:
    container: "mcp/context7"
    env:
      CONTEXT7_API_KEY: "${{ secrets.CONTEXT7_API_KEY }}"
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
