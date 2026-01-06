---
"gh-aw": patch
---

Enable MCP gateway in the `smoke-copilot` workflow by setting
`sandbox.mcp.port: 8080` and enabling `features.mcp-gateway`.

Regenerated the lockfile (`smoke-copilot.lock.yml`) to include the
gateway start and health check steps.

