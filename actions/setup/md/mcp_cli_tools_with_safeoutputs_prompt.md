<mcp-clis>
The following servers are available as CLI executables on `PATH`. Invoke them from bash when you need CLI transport. Unless a server is explicitly listed in another prompt section as an MCP tool (for example `<safe-output-tools>`), do not call these via an MCP tool interface.

__GH_AW_MCP_CLI_SERVERS_LIST__

For `mcpscripts`, prefer its native functions when they are explicitly exposed in your tool inventory. Otherwise use the CLI commands above through Bash; `mcpscripts` is a server/executable name, not an individual MCP function. A CLI example does not grant permission for its shell commands or pipelines.

For `safeoutputs`, every successful call is a real write-intent declaration - do not use it for probing, auth checks, or placeholder payloads. Use `noop` or `report_incomplete` if not ready to emit the final action.

Pass the complete JSON object inline as a single shell-quoted argument. Encode newlines inside JSON strings as `\n`, and escape JSON and shell quotes correctly. The binary comes first, so a malformed payload still runs the CLI and reports an error:
```bash
safeoutputs add_comment '{"item_number":42,"body":"### Title\n\nBody."}'
```
CLI proxying grants the mounted executable, not auxiliary commands, pipelines, file writes, or shell expansions. Do not stage or serialize payloads with helper commands unless the workflow explicitly permits them. Raw Markdown and `@filename` references are not JSON payloads.

If the sandbox rejects a command, do not repeat it or switch to an unapproved shell workaround. Use a native safe-output function only if it is explicitly exposed in your tool inventory; otherwise call `safeoutputs report_incomplete` with inline JSON containing `reason` and optional `details`. If even that call is unavailable or denied, state the blocker in your final response; do not probe with real writes.

The generated command syntax above is schema-derived from each enabled tool's final `inputSchema` and is the source of truth for required/optional parameters.
Use `<server> --help` and `<server> <tool> --help` for the same schema-derived signatures and examples before calling any command.
</mcp-clis>
