---
title: MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
sidebar:
  order: 400
---

The `gh aw mcp-server` command exposes `gh aw` CLI tools (status, compile, logs, audit) to AI agents through the Model Context Protocol. The MCP server enables AI agents to:
- Check workflow status and compile workflows
- Download and analyze workflow logs
- Investigate workflow run failures

Start the server for local CLI usage:

```bash
gh aw mcp-server
```

Or configure in for any host:
```yaml
command: gh
args: [aw, mcp-server]
```

## Available Tools

The MCP server provides these tools:

- **status** - List workflows with optional pattern filter
- **compile** - Compile workflows to GitHub Actions YAML
- **logs** - Download workflow logs (saved to `/tmp/gh-aw/aw-mcp/logs`)
- **audit** - Generate detailed workflow run report (saved to `/tmp/gh-aw/aw-mcp/logs`)

## Example Prompt

```markdown
Check all workflows in this repository:

1. Use `status` to list workflows
2. Use `logs` to get recent runs (last 5 for each workflow)
3. Use `audit` to investigate any failures
4. Generate a summary report

```

