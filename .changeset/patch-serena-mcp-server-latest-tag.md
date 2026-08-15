---
"gh-aw": patch
---

Switch the shared Serena MCP workflow container reference from `ghcr.io/github/serena-mcp-server:sha-891c160` to `ghcr.io/github/serena-mcp-server:latest` so daily compilation with `--force-refresh-container-pins` can pick up upstream security rebuilds without requiring a repo-side tag bump.
