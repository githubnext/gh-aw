---
"gh-aw": patch
---

Enable MCP gateway in the `smoke-copilot` workflow by setting `sandbox.mcp.port: 8080` and
enabling `features.mcp-gateway`. Regenerated the lockfile to include gateway start and health
check steps.

This is a non-breaking internal/tooling change (workflow configuration and lockfile regeneration).

