---
"gh-aw": patch
---

Add MCP tools list to action logs. `generatePlainTextSummary()` now writes a formatted
list of available MCP tools to action logs (via `core.info()`), improving visibility
when reviewing execution logs. Tests were added and workflows were recompiled.

