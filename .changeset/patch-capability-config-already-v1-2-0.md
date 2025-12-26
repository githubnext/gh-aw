---
"gh-aw": patch
---

Document that MCP server capability configuration already uses v1.2.0 simplified API.
Both `pkg/cli/mcp_server.go` and `pkg/awmg/gateway.go` already use the modern
`ServerOptions.Capabilities` pattern from go-sdk v1.2.0, eliminating verbose
capability construction code.

No code changes required - this changeset documents the completion of issue #7711.
