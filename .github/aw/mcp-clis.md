---
description: MCP CLI command usage guidance and JSON payload patterns
---

# MCP CLI Usage

The MCP CLI feature exposes mounted MCP servers as shell commands on `PATH` and provides a consistent command interface for tool calls.
They are enabled by `tools.cli-proxy: true`.

> **IMPORTANT**: For `safeoutputs` and `mcpscripts`, **always use the CLI commands** instead of the equivalent MCP tools. The CLI wrappers are the preferred interface — do **not** call their MCP tools directly even though they may appear in your tool list.
>
> For `safeoutputs`, treat every successful command as a real write-intent declaration. Do **not** use `safeoutputs` for exploratory probing, auth checks, placeholder payloads, repeated "try again" variants, or manual runtime experiments. When you need a safe-output, emit the final intended call once. If you are not ready to perform the real action, use `noop` or `report_incomplete` instead.
>
> For all other servers listed here, they are **only** available as CLI commands and are **not** available as MCP tools.

## How to use

Each server is a standalone executable on your `PATH`. Invoke it from bash like any other shell command:

```bash
# Call a tool — pass arguments as --name value pairs
<server-name> <tool-name> --param1 value1 --param2 value2
```

**Example** — using the `playwright` CLI:
```bash
playwright --help                                  # list all browser tools
playwright browser_navigate --url https://example.com
playwright browser_snapshot                        # capture page accessibility tree
```

**Example** — using the `safeoutputs` CLI (safe outputs) when you are ready to emit the final real action:
```bash
safeoutputs add_comment --item_number 42 --body "Analysis complete"
```

**Example** — using the `mcpscripts` CLI (mcp-scripts):
```bash
mcpscripts --help                                  # list all script tools
mcpscripts mcpscripts-gh --args "pr list --repo owner/repo --limit 5"
```

Passing multiple or complex arguments (preferred):

**Preferred approach for any tool call with multiple or complex arguments**: supply a JSON object on stdin using `.` as the sentinel. The bridge parses stdin as the argument object, preserving all native types (numbers, booleans, arrays) without shell-quoting issues.

```bash
# Full argument payload as JSON via printf pipe
printf '{"item_number":42,"body":"### Title\n\nBody paragraph one.\n\nBody paragraph two."}' \
  | safeoutputs add_comment .

# Works with any tool — just match the parameter names from <server> <tool> --help
printf '{"title":"Fix: something","body":"Details here","labels":["bug","priority-high"]}' \
  | safeoutputs create_issue .
```

If pipes are blocked by bash policy, write JSON to a file and use redirection with `.` (for example: `safeoutputs create_pull_request . < /tmp/payload.json`).

> **Why prefer JSON payload mode?**
> - Preserves native types and avoids shell quoting/escaping pitfalls
> - Supports both pipe and file-redirection input with the same `.` sentinel

## Notes

- **Prefer JSON payload mode** (`. < file` or `printf '{...}' | server tool .`) for any call with multiple arguments or complex values
- All parameters can also be passed as `--name value` pairs; boolean flags can be set with `--flag` (no value) to mean `true`
- Use `.` as the only argument to parse stdin as a JSON object (all parameters supplied at once)
- Parameter names with hyphens or underscores are interchangeable (e.g. `issue-number` and `issue_number` both work)
- Output is printed to stdout; errors are printed to stderr with a non-zero exit code
- Run the CLI commands inside a `bash` tool call — they are shell executables, not MCP tools
- These CLI commands are read-only and cannot be modified by the agent
