---
"gh-aw": patch
---

Bump the MCP gateway image to `v0.3.9` so compiled workflows use the updated `gh-aw-mcpg` container. This release moves the wazero compilation cache directory to `/tmp/gh-aw/wazero-cache/` (outside of `/tmp/gh-aw/mcp-logs/`), fixing the EACCES error when the artifact upload step tries to zip the logs directory.
