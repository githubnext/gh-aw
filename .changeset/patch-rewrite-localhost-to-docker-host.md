---
"gh-aw": patch
---

Rewrite MCP server URLs using `localhost` or `127.0.0.1` to
`host.docker.internal` when the firewall is enabled so agents running
inside firewall containers can reach host MCP servers.

This change adds `RewriteLocalhostToDocker` to `MCPConfigRenderer` and
propagates sandbox configuration through MCP renderers. The rewriting
is skipped when `sandbox.agent.disabled: true` so existing localhost
URLs are preserved when explicitly disabled.

