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

Start the server on a specific port:

```bash
gh aw mcp-server --port 3000
```

Configure in your workflow:
```yaml
---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/gh-aw-mcp.md  # Includes server setup and configuration
---

# Your workflow instructions here
```

The shared configuration (`shared/gh-aw-mcp.md`) automatically:
- Builds the gh-aw CLI
- Starts the MCP server on port 3000
- Configures HTTP transport with GITHUB_TOKEN access

### Manual HTTP Configuration

If you need custom setup:

```yaml
---
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  
  - name: Install dependencies
    run: make deps-dev
  
  - name: Build gh-aw
    run: make build
  
  - name: Start MCP server
    run: |
      ./gh-aw mcp-server --port 3000 &
      sleep 2

mcp-servers:
  gh-aw:
    type: http
    url: http://localhost:3000
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---
```

## Available Tools

The MCP server provides these tools:

- **status** - List workflows with optional pattern filter
- **compile** - Compile workflows to GitHub Actions YAML
- **logs** - Download workflow logs (saved to `/tmp/aw-mcp/logs`)
- **audit** - Generate detailed workflow run report (saved to `/tmp/aw-mcp/logs`)

See the [CLI documentation](/gh-aw/tools/cli/#mcp-server) for detailed tool parameters.

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
  - shared/gh-aw-mcp.md
---

# Weekly Workflow Audit

Check all workflows in this repository:

1. Use `status` to list workflows
2. Use `logs` to get recent runs (last 5 for each workflow)
3. Use `audit` to investigate any failures
4. Generate a summary report

Logs are in `/tmp/aw-mcp/logs` for analysis.
```

## Related Documentation

- [MCP Integration Guide](/gh-aw/guides/mcps/) - Using MCP servers in workflows
- [CLI Commands](/gh-aw/tools/cli/) - Complete CLI reference
