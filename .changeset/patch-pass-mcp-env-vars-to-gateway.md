---
"gh-aw": patch
---

Pass MCP environment variables through to the MCP gateway (awmg) so the gateway process has access to the same secrets and env vars configured in the "Setup MCPs" step. This centralizes env var collection and updates gateway step generation and tests.

Files changed (PR #8677):
- pkg/workflow/mcp_servers.go
- pkg/workflow/gateway.go
- pkg/workflow/gateway_test.go
