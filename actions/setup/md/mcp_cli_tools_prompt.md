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

### Multiline String Arguments (stdin piping)

For parameters that contain multiline content (such as `--body` in `add_comment`), use `-` as the value and pipe the content via stdin. This avoids shell quoting and escaping issues:

```bash
# Write multiline content to a file and pipe it
cat body.txt | safeoutputs add_comment --body -

# Or use a here-doc / printf for inline multiline content
printf '### Title\n\nBody paragraph one.\n\nBody paragraph two.' | safeoutputs add_comment --body -

# Works with --key=- form too
printf 'multiline\ncontent' | safeoutputs add_comment --body=-
```

> **Important**: Always use stdin piping (`--body -`) instead of command substitution (`--body "$(cat file)"`) when the content contains newlines. Command substitution can strip trailing newlines and cause other quoting problems.

### Notes

- All parameters are passed as `--name value` pairs; boolean flags can be set with `--flag` (no value) to mean `true`
- Use `-` as a value to read that parameter from stdin (useful for multiline content)
- Output is printed to stdout; errors are printed to stderr with a non-zero exit code
- Run the CLI commands inside a `bash` tool call — they are shell executables, not MCP tools
- These CLI commands are read-only and cannot be modified by the agent
</mcp-clis>
