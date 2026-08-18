---
"gh-aw": patch
---

Workflows that disable shell access no longer receive CLI-only tool instructions. When `tools.bash` is disabled (`bash: false` or `bash: []`), the `<mcp-clis>` prompt section — which tells the agent to invoke `safeoutputs` and other CLI-mounted MCP servers from bash — is omitted, so the agent is directed to the MCP tools it can actually call.

The compiler now also rejects `tools.cli-proxy: true` when bash is disabled, and strict mode requires an explicit `tools.cli-proxy: false` alongside a disabled `tools.bash`. The new `cli-proxy-false-when-bash-disabled` codemod (`gh aw fix`) adds the explicit setting to existing workflows.
