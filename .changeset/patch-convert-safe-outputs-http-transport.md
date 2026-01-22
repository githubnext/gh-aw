---
"gh-aw": patch
---

Convert the safe-outputs MCP server from stdio transport to HTTP transport. This change
follows the safe-inputs pattern and includes:

- HTTP server implementation and startup scripts for safe-outputs
- Updated MCP configuration rendering to use HTTP transport and Authorization header
- Added environment variables and startup steps for the safe-outputs server
- Tests and TOML rendering updated to match HTTP transport

This is an internal implementation change; there are no user-facing CLI breaking
changes.

