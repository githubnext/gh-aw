---
"gh-aw": major
---

Restrict MCP local workflow capabilities.

**Breaking change:** MCP `add`, `update`, `fix`, and `compile` calls are disabled by default. Enable them only for trusted clients and workflow sources by adding `--allow-write-tools` to the operator's `gh aw mcp-server` startup arguments. When `--validate-actor` is enabled, these tools also require repository write access.

MCP `mcp-inspect` now lists configuration only and rejects live `server` and `tool` filters. It never executes workflow-declared commands or connects to configured endpoints. For live discovery, review and trust the workflow before invoking `gh aw mcp inspect <workflow>` directly from the CLI.
