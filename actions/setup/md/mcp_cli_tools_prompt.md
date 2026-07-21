<mcp-clis>
The following servers are available as CLI executables on `PATH`. Invoke them from bash - they are **not** MCP tools; do not call them via an MCP tool interface.

__GH_AW_MCP_CLI_SERVERS_LIST__

For `safeoutputs` and `mcpscripts`, always use the CLI commands above.

For `safeoutputs`, every successful call is a real write-intent declaration - do not use it for probing, auth checks, or placeholder payloads. Use `noop` or `report_incomplete` if not ready to emit the final action.

For multiple or complex arguments, pipe a JSON object on stdin using `.` as the sentinel:
```bash
printf '{"item_number":42,"body":"### Title\n\nBody."}' | safeoutputs add_comment .
# or write to a file: safeoutputs create_pull_request . < /tmp/payload.json
```

The generated command syntax above is schema-derived from each enabled tool's final `inputSchema` and is the source of truth for required/optional parameters.
Use `<server> --help` and `<server> <tool> --help` for the same schema-derived signatures and examples before calling any command.
</mcp-clis>
