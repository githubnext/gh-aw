---
"gh-aw": patch
---

Fix `mcp inspect` to apply imports before extracting MCP configurations

The `mcp inspect` command now correctly processes imports from workflow frontmatter before extracting MCP server configurations. This ensures that MCP servers defined in imported files are visible to the inspect command, matching the behavior of the workflow compiler.
