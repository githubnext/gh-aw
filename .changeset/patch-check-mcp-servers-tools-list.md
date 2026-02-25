---
"gh-aw": patch
---

Ensure `check_mcp_servers.sh` uses `tools/list` instead of `ping` so the readiness probe waits for the backend containers to become available.
