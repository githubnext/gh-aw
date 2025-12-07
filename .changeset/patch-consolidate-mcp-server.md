---
"gh-aw": patch
---

Consolidate duplicate MCP server implementations by using a single shared core
implementation and removing the duplicate `mcp_server.cjs` code.

This is an internal refactor that reduces duplicated code and simplifies
maintenance for MCP server transports (HTTP and stdio).

