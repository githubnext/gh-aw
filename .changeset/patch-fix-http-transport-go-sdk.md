---
"gh-aw": patch
---

Fix HTTP transport usage of go-sdk

Fixed the MCP server HTTP transport implementation to use the correct `NewStreamableHTTPHandler` API from go-sdk instead of the deprecated SSE handler. Also added request/response logging middleware and changed configuration validation errors to warnings to allow server startup in test environments.
