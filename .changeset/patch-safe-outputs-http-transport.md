---
"gh-aw": patch
---

Migrate the `safe-outputs` MCP server from stdio transport to an HTTP transport and
update the generated workflow steps to start the HTTP server before the agent.

This change adds the HTTP server implementation and startup scripts, replaces the
stdio-based MCP server configuration with an HTTP-based configuration, and updates
environment variables and MCP host resolution to support the new transport.
