---
# Svelte MCP Server
# Remote HTTP MCP server for Svelte 5 and SvelteKit documentation
#
# No authentication required - public service
# Documentation: https://svelte.dev/docs/mcp/overview
#
# Available tools:
#   - list-sections: Lists all available documentation sections
#   - get-documentation: Retrieves full and up-to-date Svelte 5 and SvelteKit documentation
#   - svelte-autofixer: Static analysis suggestions for generated code
#   - playground-link: Generates a playground link to test generated code
#   - svelte_definition: Accesses detailed Svelte 5 and SvelteKit definitions
#
# Usage:
#   imports:
#     - shared/mcp/svelte.md

mcp-servers:
  svelte:
    url: "https://mcp.svelte.dev/mcp"
    allowed:
      - list-sections
      - get-documentation
      - svelte-autofixer
      - playground-link
      - svelte_definition
---
