---
"gh-aw": patch
---

Always start the MCP gateway for behavior-defined engines that declare no `behaviors.mcp.config-path`.

Engines such as Aider (`mcp: false`) consume MCP-backed tools through the CLI proxy and do not
declare an MCP config path. The compiler previously skipped rendering the gateway configuration
for these engines, so `start_mcp_gateway.cjs` was never invoked and the `awmg-mcpg` container was
never started — while AWF was still instructed to attach it to the internal network. This made the
firewall abort with `Failed to connect container "awmg-mcpg" to network "awf-net"`, terminating the
engine before it ran.

The MCP configuration is now rendered using the default MCP servers path when the engine declares
no explicit `config-path`, so the gateway container always starts.
