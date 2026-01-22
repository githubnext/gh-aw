---
"gh-aw": patch
---

Convert the `safe-outputs` MCP server from stdio transport to an HTTP transport.

This change updates the MCP server implementation, startup steps, and generated
MCP configuration to use an HTTP server with an Authorization header instead of
stdio. Tests and workflow lock files were updated accordingly.
