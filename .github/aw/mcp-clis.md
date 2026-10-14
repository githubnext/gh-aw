---
description: MCP CLI command usage guidance and JSON payload patterns
---

# MCP CLI Usage

MCP CLI exposes mounted MCP servers as shell commands on `PATH`. Enabled by `tools.cli-proxy: true`.

> **IMPORTANT**: Use the configured transport. Mounted CLI commands are available through the shell; native functions are an option only when explicitly exposed in the engine's tool inventory. CLI examples do not grant additional shell permissions.
>
> For `safeoutputs`, treat every successful command as a real write-intent declaration. Do **not** use it for exploratory probing, auth checks, placeholder payloads, retries with variants, or runtime experiments. Emit the final intended call once. If not ready, use `noop` or `report_incomplete`.
>
> All other servers listed here are **only** available as CLI commands, not MCP tools.

## How to use

Each server is a standalone executable on `PATH`. Invoke from bash with `--name value` pairs:

```bash
<server-name> <tool-name> --param1 value1 --param2 value2
```

**Examples:**
```bash
github --help
github issue_read --method get --owner org --repo repo --issue_number 42

safeoutputs add_comment '{"item_number":42,"body":"Analysis complete"}'

mcpscripts --help
mcpscripts mcpscripts-gh --args "pr list --repo owner/repo --limit 5"
```

For safe-output calls, prefer the complete JSON object inline as one single-quoted shell argument. Inside JSON strings, encode newlines as `\n`, double quotes as `\"`, backslashes as `\\`, and apostrophes as `\u0027` so they do not end the shell quote. Unicode can remain literal:

```bash
safeoutputs add_comment '{"item_number":42,"body":"### Result\n\nIt\u0027s ready.\nQuoted text: \"hello\".\nLiteral path: C:\\temp\\file.txt"}'
```

Stdin is also supported: `safeoutputs <tool_name> .` reads a complete JSON object from stdin. Prefer inline JSON without auxiliary tools, pipelines, temporary files, or shell expansion. Transport support does not grant permission for additional commands or redirections; never bypass a denial with an unapproved workaround.

## Notes

- **Prefer inline JSON** for safe-output calls, including multiline bodies and structured arguments
- Parameters also accept `--name value` pairs; boolean flags use `--flag` (no value) for `true`
- `.` as sole argument parses stdin as a JSON object
- Hyphens and underscores in parameter names are interchangeable (`issue-number` == `issue_number`)
- Output goes to stdout; errors to stderr with non-zero exit
- Run inside a `bash` tool call — these are shell executables, not MCP tools
- Read-only; cannot be modified by the agent
