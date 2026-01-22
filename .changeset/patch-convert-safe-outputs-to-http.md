---
"gh-aw": patch
---

Convert the `safe-outputs` MCP server from stdio/container transport to an HTTP-based transport and update generated workflow steps to start the HTTP server before the agent. This migrates to a stateful HTTP service, removes stdio/container fields from the generated MCP configuration, and exposes `GH_AW_SAFE_OUTPUTS_PORT` and `GH_AW_SAFE_OUTPUTS_API_KEY` for MCP gateway resolution.

This is an internal/tooling change and does not change public CLI APIs.

