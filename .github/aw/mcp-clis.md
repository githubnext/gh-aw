---
description: MCP CLI command usage guidance and JSON payload patterns
---

# MCP CLI Usage

MCP CLI exposes mounted MCP servers as shell commands on `PATH`. Enabled by `tools.cli-proxy: true`.

> **IMPORTANT**: For `safeoutputs` and `mcpscripts`, **always use the CLI commands** instead of the equivalent MCP tools — do **not** call their MCP tools directly even if they appear in your tool list.
>
> For `safeoutputs`, treat every successful command as a real write-intent declaration. Do **not** use it for exploratory probing, auth checks, placeholder payloads, retries with variants, or runtime experiments. Emit the final intended call once. If not ready, use `noop` or `report_incomplete`.
>
> All other servers listed here are **only** available as CLI commands, not MCP tools.

## How to use

Each server is a standalone executable on `PATH`. Invoke it from bash:

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

Supply a JSON object on stdin using `.` as the sentinel. The bridge parses stdin as the argument object, preserving native types (numbers, booleans, arrays) without shell-quoting issues.

```bash
# Full argument payload as JSON via printf pipe
printf '{"item_number":42,"body":"### Title\n\nBody paragraph one.\n\nBody paragraph two."}' \
  | safeoutputs add_comment .

# Works with any tool — just match the parameter names from <server> <tool> --help
printf '{"title":"Fix: something","body":"Details here","labels":["bug","priority-high"]}' \
  | safeoutputs create_issue .
```

If pipes are blocked, write JSON to a file and use redirection with `.` (e.g. `safeoutputs create_pull_request . < /tmp/payload.json`).

## Notes

- **Prefer JSON payload mode** (`. < file` or `printf '{...}' | server tool .`) for multi-argument or complex calls
- Parameters also accept `--name value` pairs; boolean flags use `--flag` (no value) for `true`
- `.` as sole argument parses stdin as a JSON object
- Hyphens and underscores in parameter names are interchangeable (`issue-number` == `issue_number`)
- Output goes to stdout; errors to stderr with non-zero exit
- Run inside a `bash` tool call — these are shell executables, not MCP tools
- Read-only; cannot be modified by the agent
