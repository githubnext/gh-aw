---
title: MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
---

The `mcp-server` command exposes gh-aw CLI tools (status, compile, logs, audit) to AI agents through the Model Context Protocol. This allows agents to manage workflows while keeping GitHub tokens secure.

## Why Use MCP?

The MCP server enables AI agents to:
- Check workflow status and compile workflows
- Download and analyze workflow logs
- Investigate workflow run failures

**Key Security Benefit**: GitHub tokens remain isolated in the MCP server process and are never exposed to the agentic workflow process.

## Configuration

### Stdio Transport (Local Development)

Start the server for local CLI usage:

```bash
gh aw mcp-server
```

Configure in your MCP client:
```yaml
mcp-servers:
  gh-aw:
    command: gh
    args: [aw, mcp-server]
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### HTTP Transport (Workflows)

For workflows, use the shared configuration that builds and starts the MCP server with GITHUB_TOKEN:

```yaml
---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/mcp/gh-aw.md
---

# Your workflow instructions here
```

The shared configuration (`shared/mcp/gh-aw.md`) contains:

**Steps** (run with GITHUB_TOKEN):
```yaml
steps:
  - name: Install gh-aw
    run: gh extension install githubnext/gh-aw   
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
  - name: Start MCP server
    run: gh aw mcp-server --port 8765 &
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
```

**MCP Server Configuration** (HTTP transport, no secret):
```yaml
mcp-servers:
  gh-aw:
    type: http
    url: http://localhost:8765
```

This pattern keeps the GITHUB_TOKEN isolated in the server process while the workflow accesses it through HTTP transport.

## Available Tools

The MCP server provides these tools:

- **status** - List workflows with optional pattern filter
- **compile** - Compile workflows to GitHub Actions YAML
- **logs** - Download workflow logs (saved to `/tmp/gh-aw/aw-mcp/logs`)
- **audit** - Generate detailed workflow run report (saved to `/tmp/gh-aw/aw-mcp/logs`)

## Example Workflow

```aw
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/mcp/gh-aw.md
---

# Weekly Workflow Audit

Check all workflows in this repository:

1. Use `status` to list workflows
2. Use `logs` to get recent runs (last 5 for each workflow)
3. Use `audit` to investigate any failures
4. Generate a summary report

Logs are in `/tmp/gh-aw/aw-mcp/logs` for analysis.
```

