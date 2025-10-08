---
"gh-aw": minor
---

Add mcp-server command to expose CLI tools via Model Context Protocol

This PR adds a new `mcp-server` command that implements a Model Context Protocol (MCP) server using the go-mcp SDK. The server exposes existing CLI commands (status, compile, logs, audit) as MCP tools, supporting both stdio and HTTP/SSE transports. This enables integration with AI assistants and other MCP clients, allowing them to interact with gh-aw workflows programmatically.
