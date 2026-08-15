---
"gh-aw": patch
---

Fix `assign_to_agent` concurrency handling by isolating per-handler assignment state, making max-slot enforcement atomic, and serializing MCP stdin dispatch to prevent overlapping tool execution.
