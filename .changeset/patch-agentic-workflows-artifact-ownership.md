---
"gh-aw": patch
---

Run the built-in agentic-workflows MCP container with the runner's UID and GID so private log downloads and audit files remain readable by host secret redaction. Create a fresh, private runner-temp home for each setup with a dedicated writable mount and reject symlinked parent paths, avoiding stale or root-owned homes on shared runners. Preserve owner-only permissions and fail-closed artifact removal.
