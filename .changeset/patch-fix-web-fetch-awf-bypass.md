---
"gh-aw": patch
---

Enforce AWF domain allowlist on the `mcp/fetch` container to close the web-fetch bypass.

The `mcp/fetch` container ran outside AWF's network namespace, allowing the web-fetch
MCP tool to reach any URL regardless of the workflow's `network.allowed` restrictions.

The fix passes `--allowed-domains` to the `mcp-server-fetch` entrypoint when AWF is
active with a non-wildcard domain list. This mirrors the same allowlist AWF enforces
for the agent container so both enforcement layers agree on which destinations are
reachable.

No change when `network.allowed: ["*"]` (unrestricted) or when the AWF firewall is
not enabled.
