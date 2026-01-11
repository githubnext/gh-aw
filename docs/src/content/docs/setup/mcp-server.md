---
title: MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
sidebar:
  order: 400
---

The `gh aw mcp-server` command exposes CLI tools (status, compile, logs, audit, update, add, mcp-inspect) to AI agents through the Model Context Protocol.

Start the server:
```bash wrap
gh aw mcp-server
```

Or configure for any Model Context Protocol (MCP) host:
```yaml wrap
command: gh
args: [aw, mcp-server]
```

> [!TIP]
> Use in agentic workflows by adding `agentic-workflows:` to your workflow's `tools:` section. See [Using as Agentic Workflows Tool](#using-as-agentic-workflows-tool).

## Configuration Options

### HTTP Server Mode

Run with HTTP/SSE transport using `--port`:

```bash wrap
gh aw mcp-server --port 8080
```

## Configuring with GitHub Copilot Agent

Configure GitHub Copilot Agent to use gh-aw MCP server:

```bash wrap
gh aw init
```

This creates `.github/workflows/copilot-setup-steps.yml` that sets up Go, GitHub CLI, and gh-aw extension before agent sessions start, making workflow management tools available to the agent. MCP server integration is enabled by default. Use `gh aw init --no-mcp` to skip MCP configuration.

## Configuring with Copilot CLI

To add the MCP server in the interactive Copilot CLI session, start `copilot` and run:

```
/mcp add github-agentic-workflows gh aw mcp-server
```

## Configuring with VS Code

Configure VS Code Copilot Chat to use gh-aw MCP server:

```bash wrap
gh aw init
```

This creates `.vscode/mcp.json` and `.github/workflows/copilot-setup-steps.yml`. MCP server integration is enabled by default. Use `gh aw init --no-mcp` to skip MCP configuration.

Alternatively, create `.vscode/mcp.json` manually:

```json wrap
{
  "servers": {
    "github-agentic-workflows": {
      "command": "gh",
      "args": ["aw", "mcp-server"],
      "cwd": "${workspaceFolder}"
    }
  }
}
```

Reload VS Code after making changes.

## Available Tools

The MCP server provides:
- **status**: List workflows with pattern filter
- **compile**: Generate GitHub Actions YAML
- **logs**: Download with timeout handling and continuation
- **audit**: Generate report to `/tmp/gh-aw/aw-mcp/logs`. Accepts run IDs, workflow run URLs, job URLs, and step-level URLs for precise failure analysis
- **update**: Update workflows with support for major version updates and force flag
- **add**: Install workflows from remote repositories
- **mcp-inspect**: Inspect servers and validate secrets

### Logs Tool Features

**Timeout and Continuation**: Uses 50-second timeout for large runs. Returns partial results with `continuation` field containing `before_run_id` to resume fetching.

**Output Size Guardrail**: Default 12,000 tokens (~48KB) limit. Customize with `max_tokens` parameter. When triggered, provides jq queries for filtering (e.g., `'.runs | map(select(.conclusion == "failure"))'`).

**Large Output Handling**: Outputs exceeding 16,000 tokens (~64KB) are written to `/tmp/gh-aw/safe-outputs/` with file location and schema description returned.

## Using as Agentic Workflows Tool

Enable in workflow frontmatter:

```yaml wrap
---
permissions:
  actions: read  # Required for agentic-workflows tool
tools:
  agentic-workflows:
---

Check workflow status, download logs, and audit failures.
```

> [!CAUTION]
> Required Permission
> The `agentic-workflows` tool requires `actions: read` permission to access GitHub Actions workflow logs and run data.

