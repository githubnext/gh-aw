<mcp-clis>
The following servers are available as CLI executables on `PATH`. Invoke them from bash when you need CLI transport. Unless a server is explicitly listed in another prompt section as an MCP tool (for example `<safe-output-tools>`), do not call these via an MCP tool interface.

__GH_AW_MCP_CLI_SERVERS_LIST__

For `mcpscripts`, always use the CLI commands above.

For `safeoutputs`, every successful call is a real write-intent declaration - do not use it for probing, auth checks, or placeholder payloads. Use `noop` or `report_incomplete` if not ready to emit the final action.

Before using CLI transport, check `command -v safeoutputs`. If it is missing from `PATH`, restore the mounted wrappers with `export PATH="${RUNNER_TEMP}/gh-aw/mcp-cli/bin:$PATH"` and check again. If still unavailable, use the configured native safe-output MCP tools when exposed; do not invent an inline-JSON shell fallback or repeatedly retry a missing command. CLI-only engines must stop and explain the infrastructure failure.

For multiple or complex arguments, pass the JSON object inline as a single quoted argument — the binary comes first, so a malformed command still runs the CLI and reports an error:
```bash
safeoutputs add_comment '{"item_number":42,"body":"### Title\n\nBody."}'
```
Alternatively, pipe a JSON object on stdin using `.` as the sentinel (prefer this only for large bodies built from files):
```bash
printf '{"item_number":42,"body":"### Title\n\nBody."}' | safeoutputs add_comment .
# or read from a file: safeoutputs create_pull_request . < /tmp/payload.json
```
When piping, make sure the `|` is actually present: `printf '{...}' safeoutputs add_comment .` (no pipe) prints the JSON and exits 0 without ever running the CLI, so nothing is emitted.

**Multi-line or long `body` content:** do NOT embed the body in inline JSON or a long quoted argument. Write it to a temp file with a quoted heredoc, then pass `--body .` with stdin redirection. The CLI reads the file and encodes the request; this does not require `jq`, Python, or `apply_patch`:
```bash
cat <<'EOF' > /tmp/gh-aw/body.md
Title

Multi-line body content goes here.
EOF
safeoutputs create_discussion --title "My title" --body . < /tmp/gh-aw/body.md
```
If a shell command is rejected for containing expansion patterns, do not retry the same command — switch to the heredoc + `--body .` pattern above.

To inject an entire local file as the `body` field without re-embedding its content in the model context:
```bash
safeoutputs update_discussion --discussion_number "$DISCUSSION_NUMBER" --body . < discussion-body.md
safeoutputs add_comment --item_number 42 --body . < /tmp/gh-aw/body.md
```
The `.` after `--body` selects per-field stdin mode. Without `--body .`, raw Markdown is not a JSON payload and will be rejected. If using `jq` for a complex JSON payload, first check `command -v jq`; never assume optional shell tools are installed.

The generated command syntax above is schema-derived from each enabled tool's final `inputSchema` and is the source of truth for required/optional parameters.
Use `<server> --help` and `<server> <tool> --help` for the same schema-derived signatures and examples before calling any command.
</mcp-clis>
