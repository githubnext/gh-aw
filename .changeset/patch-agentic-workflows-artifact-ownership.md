---
"gh-aw": patch
---

Run the built-in agentic-workflows MCP container with the runner's UID and GID so private log downloads and audit files remain readable by host secret redaction. Preserve owner-only permissions and fail-closed artifact removal.
