---
"gh-aw": patch
---

Ensure `check_mcp_servers.sh` runs the full MCP handshake (ping → initialize → tools/list) with the session ID so the readiness probe waits for backend containers.
