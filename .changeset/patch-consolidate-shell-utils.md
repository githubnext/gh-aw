---
"gh-aw": patch
---

Consolidate shell escaping utilities into `shell.go` and remove the duplicate helpers in `mcp_utilities.go` so the generator and tests use a single source of truth.
