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

To inject an entire local file as the `body` field without re-embedding its content in the model context, use `jq -Rs`:
```bash
jq -Rs --arg discussion_number "$DISCUSSION_NUMBER" \
  '{discussion_number: ($discussion_number|tonumber), body: .}' \
  discussion-body.md \
  | safeoutputs update_discussion .
```
`jq -Rs` reads the file as a raw string (`-R`) and slurps it into a single JSON string value (`-s`), so `body` is always a valid JSON field. Piping `cat file | safeoutputs ...` does not populate `body` and will be rejected.

The generated command syntax above is schema-derived from each enabled tool's final `inputSchema` and is the source of truth for required/optional parameters.
Use `<server> --help` and `<server> <tool> --help` for the same schema-derived signatures and examples before calling any command.
</mcp-clis>
