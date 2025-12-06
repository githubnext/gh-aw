---
"gh-aw": patch
---

Implement a local MCP HTTP transport layer and remove the `@modelcontextprotocol/sdk` dependency.

Adds `mcp_logger.cjs`, `mcp_server.cjs`, and `mcp_http_transport.cjs` plus unit and integration tests. Internal refactor and tooling change only; no public CLI breaking changes.
