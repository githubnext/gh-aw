---
"gh-aw": patch
---

Wire up MCP config schema validation so malformed MCP server configs (for example invalid `container`, malformed `mounts`, invalid `env` keys, and unknown top-level properties) now fail validation instead of passing silently.
