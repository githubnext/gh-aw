---
"gh-aw": patch
---

Added opt-in MCP CLI mounting via `tools.mount-as-clis: true`, which exposes eligible MCP servers as local CLI wrappers on `PATH` and updates prompt/config wiring so agents use those wrappers. The `github` MCP server remains a normal MCP tool, while `safeoutputs` and `mcpscripts` are included in CLI mounting when enabled.
