---
title: MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
sidebar:
  order: 400
---

The `gh aw mcp-server` command exposes `gh aw` CLI tools (status, compile, logs, audit, mcp-inspect) to AI agents through the Model Context Protocol, enabling agents to check workflow status, compile workflows, download logs, investigate failures, and inspect MCP servers.

:::tip
Enable this MCP server in agentic workflows by adding `agentic-workflows:` to the `tools:` section in your workflow frontmatter. See [Using as Agentic Workflows Tool](#using-as-agentic-workflows-tool) for details.
:::

Start the server for local CLI usage:

```bash
gh aw mcp-server
```

Or configure in for any host:
```yaml
command: gh
args: [aw, mcp-server]
```

## Configuration Options

### Using a Custom Command Path

Use the `--cmd` flag to specify a custom path to the gh-aw binary instead of using the default `gh aw` command:

```bash
gh aw mcp-server --cmd ./gh-aw
```

Use this for local development builds, CI/CD workflows with specific versions, or environments without the gh CLI extension.

Example in an agentic workflow:
```yaml
steps:
  - name: Build gh-aw
    run: make build
  - name: Start MCP server
    run: |
      set -e
      ./gh-aw mcp-server --cmd ./gh-aw --port 8765 &
      MCP_PID=$!
      sleep 2
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "MCP server failed to start"
        exit 1
      fi
```

### HTTP Server Mode

Use the `--port` flag to run the server with HTTP/SSE transport instead of stdio:

```bash
gh aw mcp-server --port 8080
```

## Configuring with Copilot CLI

The GitHub Copilot CLI can use the gh-aw MCP server to access workflow management tools.

Use the `/mcp` command in Copilot CLI to add the MCP server:

```bash
/mcp add github-agentic-workflows gh aw mcp-server
```

This registers the server with Copilot CLI, making workflow management tools available in your terminal sessions.

## Configuring with VS Code

VS Code can use the gh-aw MCP server through the Copilot Chat extension.

Create or update `.vscode/mcp.json` in your repository:

```json
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

:::note
The `${workspaceFolder}` variable automatically resolves to your current workspace directory in VS Code. Use this for development builds where the `gh-aw` binary is in your project root.
:::

After adding the configuration, reload VS Code or restart the Copilot Chat extension.

## Available Tools

The MCP server provides **status** (list workflows with pattern filter), **compile** (generate GitHub Actions YAML), **logs** (download with timeout handling and continuation), **audit** (generate report to `/tmp/gh-aw/aw-mcp/logs`), and **mcp-inspect** (inspect servers and validate secrets).

### Logs Tool Features

**Timeout and Continuation:**

The logs tool uses a 50-second default timeout to prevent MCP server timeouts when downloading large workflow runs. When a timeout occurs, the tool returns partial results with a `continuation` field containing parameters to resume fetching:

```json
{
  "summary": { "total_runs": 5 },
  "runs": [ ... ],
  "continuation": {
    "message": "Timeout reached. Use these parameters to continue fetching more logs.",
    "workflow_name": "weekly-research",
    "count": 100,
    "before_run_id": 12341,
    "timeout": 50
  }
}
```

Agents can detect incomplete data by checking for the `continuation` field and make follow-up calls with the provided `before_run_id` to fetch remaining logs.

**Output Size Guardrail:**

The logs tool includes a token-based output guardrail (default: 12,000 tokens, ~48KB) to prevent overwhelming responses. When output exceeds the limit, the tool returns a structured response with:

- Warning message explaining the token limit
- Complete JSON schema of the LogsData structure
- Suggested jq queries for common filtering scenarios

The limit can be customized using the `max_tokens` parameter:

```json
{
  "name": "logs",
  "arguments": {
    "count": 100,
    "max_tokens": 20000
  }
}
```

When the guardrail triggers, use the provided jq queries to filter the data. For example, to get only failed runs, use `jq: '.runs | map(select(.conclusion == "failure"))'`.

**Large Output Handling:**

When tool outputs exceed 16,000 tokens (~64KB), the MCP server automatically writes content to `/tmp/gh-aw/safe-outputs/` and returns a JSON response with file location and schema description:

```json
{
  "filename": "bb28168fe5604623b804546db0e8c90eaf9e8dcd0f418761787d5159198b4fd8.json",
  "description": "[{id, name, data}] (2000 items)"
}
```

Schema descriptions format: JSON arrays as `[{key1, key2}] (N items)`, objects as `{key1, key2, ...} (N keys)`, and text as `text content`.

## Example Prompt

```markdown
Check all workflows: use `status` to list workflows, `logs` for recent runs, `audit` for failures, then generate a summary report.
```

## Using as Agentic Workflows Tool

The MCP server is available as a builtin tool called `agentic-workflows` in agentic workflows:

```yaml
---
tools:
  agentic-workflows:  # Enables status, compile, logs, audit, and mcp-inspect tools
---

Check workflow status, inspect MCP servers, download recent logs, and audit any failures.
```

