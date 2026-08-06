---
"gh-aw": patch
---

Prevent MCP server processes from calling `core.setSecret` by separating MCP-safe git authentication environment construction from GitHub Actions secret masking.
