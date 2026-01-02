---
"gh-aw": patch
---

Enable MCP gateway for smoke-copilot-no-firewall workflow

Enables the MCP gateway (`awmg`) so MCP server calls are routed through a centralized
HTTP proxy for the `smoke-copilot-no-firewall` workflow. Adds `features.mcp-gateway: true`
and a `sandbox.mcp` block with the gateway command and port.

This is an internal workflow/configuration change (patch).

---
summary: "Enable MCP gateway (awmg) in smoke-copilot-no-firewall workflow"

