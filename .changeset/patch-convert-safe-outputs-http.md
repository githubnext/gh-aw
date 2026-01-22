---
"gh-aw": patch
---

Convert safe-outputs MCP server to HTTP transport and update generated
startup steps and MCP configuration to use HTTP (Authorization header,
port, and URL). This is an internal implementation change that moves the
safe-outputs MCP server from stdio to an HTTP transport and updates the
workflow generation to start and configure the HTTP server.

Changes include:
- New HTTP server JavaScript and startup scripts for safe-outputs
- Updated MCP config rendering to use `type: http`, `url`, and `headers`
- Workflow step outputs for `port` and `api_key` and changes to env vars

No public CLI behavior or user-facing flags were changed; this is an
internal/backend change and therefore marked as a `patch`.

