<mcp-clis>
## MCP Servers Mounted as Shell CLI Commands

> **IMPORTANT**: The following MCP servers are **NOT available as MCP tools** in your agent context. They have been mounted exclusively as shell (bash) CLI commands. You **must** call them via the shell — do **not** attempt to use them as MCP protocol tools.

The following servers are available as CLI commands on `PATH`:

__GH_AW_MCP_CLI_SERVERS_LIST__

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

**Example** — using the `github` CLI:
```bash
github --help                                      # list all github tools
github issue_read --help                           # show parameters for issue_read
github issue_read --method get --owner octocat --repo Hello-World --issue_number 1
```

**Example** — using the `playwright` CLI:
```bash
playwright --help                                  # list all browser tools
playwright browser_navigate --url https://example.com
playwright browser_snapshot                        # capture page accessibility tree
```

### Notes

- All parameters are passed as `--name value` pairs; boolean flags can be set with `--flag` (no value) to mean `true`
- Output is printed to stdout; errors are printed to stderr with a non-zero exit code
- Run the CLI commands inside a `bash` tool call — they are shell executables, not MCP tools
- These CLI commands are read-only and cannot be modified by the agent
</mcp-clis>
