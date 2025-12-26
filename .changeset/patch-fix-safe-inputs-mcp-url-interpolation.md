---
"gh-aw": patch
---

Fix safe inputs MCP URL variable interpolation in Copilot workflows by
quoting the HERE document delimiter so backslashes are preserved. This
prevents bash from stripping escapes like `\${GH_AW_SAFE_INPUTS_PORT}`
when writing the MCP config file for Copilot, avoiding invalid URL
errors at runtime.

Files changed: `pkg/workflow/mcp_renderer.go` (quoted heredoc delimiter),
and corresponding tests updated to expect the quoted delimiter.

