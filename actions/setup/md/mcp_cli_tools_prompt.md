<mcp-clis>
## MCP Servers Mounted as Shell CLI Commands

The following servers are available as CLI commands on `PATH`:

__GH_AW_MCP_CLI_SERVERS_LIST__

> **IMPORTANT**: For `safeoutputs` and `mcpscripts`, **always use the CLI commands** listed above instead of the equivalent MCP tools. The CLI wrappers are the preferred interface — do **not** call their MCP tools directly even though they may appear in your tool list.
>
> For all other servers listed here, they are **only** available as CLI commands and are **not** available as MCP tools.

### How to Use

Each server is a standalone executable on your `PATH`. Invoke it from bash like any other shell command:

```bash
# Discover what tools a server provides
<server-name> --help

# Get detailed help for a specific tool (description + parameters)
<server-name> <tool-name> --help

# Call a tool — pass arguments as --name value pairs
<server-name> <tool-name> --param1 value1 --param2 value2
```

**Example** — using the `playwright` CLI:
```bash
playwright --help                                  # list all browser tools
playwright browser_navigate --url https://example.com
playwright browser_snapshot                        # capture page accessibility tree
```

**Example** — using the `safeoutputs` CLI (safe outputs):
```bash
safeoutputs --help                                 # list all safe-output tools
safeoutputs add_comment --body "Analysis complete"
safeoutputs upload_artifact --path "report.json"
```

**Example** — using the `mcpscripts` CLI (mcp-scripts):
```bash
mcpscripts --help                                  # list all script tools
mcpscripts mcpscripts-gh --args "pr list --repo owner/repo --limit 5"
```

### Multiline and Multi-Argument Payloads (JSON stdin)

**Preferred approach for any tool call with multiple or complex arguments**: supply a JSON object on stdin using `.` as the sentinel. The bridge parses stdin as the argument object, preserving all native types (numbers, booleans, arrays) without shell-quoting issues.

```bash
# Full argument payload as JSON via printf pipe
printf '{"issue_number":42,"body":"### Title\n\nBody paragraph one.\n\nBody paragraph two."}' \
  | safeoutputs add_comment .

# Works with any tool — just match the parameter names from <server> <tool> --help
printf '{"title":"Fix: something","body":"Details here","labels":["bug","priority-high"]}' \
  | safeoutputs create_issue .
```

**When pipes are blocked by the bash security policy**, write the payload to a file first and use **file redirection** with the `.` sentinel instead:

```bash
# Step 1 — write the JSON payload to a file using the Write tool or a bash heredoc
# Step 2 — redirect the file into the CLI command using '<'
safeoutputs create_pull_request . < /tmp/payload.json

# This is equivalent to piping but does not require a separate command before '|'
safeoutputs add_comment . < /tmp/comment.json
```

> **Why prefer JSON payload mode?**
> - Single operation for any number of arguments — no repeated `--key value` flags
> - Native types (integers, booleans, arrays) are preserved exactly as specified
> - No shell quoting or escaping needed for newlines, quotes, or special characters
> - Agents can construct the payload as a structured object before emitting the command
> - File redirection (`< file`) works even when pipes (`|`) are restricted

Key normalisation rules apply: parameter names with hyphens or underscores are interchangeable (e.g. `issue-number` and `issue_number` both work).

> **Important**: Do **not** use `--body - < file` to pass a multi-field JSON payload — `--body -` reads stdin as a raw string and will pass the entire JSON object as the body. Use `. < file` (dot sentinel) to parse stdin as JSON and distribute all fields across their respective parameters.

### Single-Parameter stdin Substitution

For the case where only **one** parameter needs multiline content, use `-` as its value:

```bash
# Write multiline content to a file and redirect it
safeoutputs add_comment --issue_number 42 --body - < body.txt

# Or use printf for inline multiline content
printf '### Title\n\nBody paragraph one.\n\nBody paragraph two.' \
  | safeoutputs add_comment --issue_number 42 --body -

# Works with --key=- form too
printf 'multiline\ncontent' | safeoutputs add_comment --issue_number 42 --body=-
```

> **Important**: Always use stdin substitution (`--body -`) instead of command substitution (`--body "$(cat file)"`) when the content contains newlines. Command substitution can strip trailing newlines and cause other quoting problems.

### Notes

- **Prefer JSON payload mode** (`. < file` or `printf '{...}' | server tool .`) for any call with multiple arguments or complex values
- All parameters can also be passed as `--name value` pairs; boolean flags can be set with `--flag` (no value) to mean `true`
- Use `.` as the only argument to parse stdin as a JSON object (all parameters supplied at once)
- Use `-` as a single value to read one parameter from stdin (single-field substitution)
- Output is printed to stdout; errors are printed to stderr with a non-zero exit code
- Run the CLI commands inside a `bash` tool call — they are shell executables, not MCP tools
- These CLI commands are read-only and cannot be modified by the agent
</mcp-clis>
