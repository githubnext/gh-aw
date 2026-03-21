---
"gh-aw": patch
---

Remove session management from the safe outputs MCP HTTP server and make it stateless-only so MCP gateway `tools/list` works without requiring `Mcp-Session-Id` or `--stateless` workarounds.
