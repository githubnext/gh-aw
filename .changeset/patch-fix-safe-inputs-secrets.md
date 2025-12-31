---
"gh-aw": patch
---

Ensure safe-inputs MCP server start step receives tool secrets via an
`env:` block so the MCP server process inherits the correct environment.
Removes redundant `export` statements in the start script that attempted
to export variables that were not present in the step environment.

Fixes passing of secrets like `GH_AW_GH_TOKEN` to the MCP server process.

