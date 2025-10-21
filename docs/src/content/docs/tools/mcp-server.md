---
title: MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
sidebar:
  order: 400
---

The `gh aw mcp-server` command exposes `gh aw` CLI tools (status, compile, logs, audit, mcp-inspect) to AI agents through the Model Context Protocol. The MCP server enables AI agents to:
- Check workflow status and compile workflows
- Download and analyze workflow logs
- Investigate workflow run failures
- Inspect MCP servers in workflows

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

This is useful when:
- Running a local build of gh-aw for development
- Using a specific version of gh-aw in CI/CD workflows
- Running the MCP server in environments where the gh CLI extension is not available

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

## Available Tools

The MCP server provides these tools:

- **status** - List workflows with optional pattern filter
- **compile** - Compile workflows to GitHub Actions YAML
- **logs** - Download workflow logs with automatic timeout handling and continuation support
- **audit** - Generate detailed workflow run report (saved to `/tmp/gh-aw/aw-mcp/logs`)
- **mcp-inspect** - Inspect MCP servers in workflows and validate secrets

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

**Large Output Handling:**

When tool outputs exceed 16,000 tokens (~64KB), the MCP server automatically writes content to `/tmp/gh-aw/safe-outputs/` and returns a JSON response with file location and schema description:

```json
{
  "filename": "bb28168fe5604623b804546db0e8c90eaf9e8dcd0f418761787d5159198b4fd8.json",
  "description": "[{id, name, data}] (2000 items)"
}
```

Schema descriptions help agents understand data structure:
- JSON arrays: `[{key1, key2}] (N items)`
- JSON objects: `{key1, key2, ...} (N keys)`
- Text content: `text content`

## Example Prompt

```markdown
Check all workflows in this repository:

1. Use `status` to list workflows
2. Use `logs` to get recent runs (last 5 for each workflow)
3. Use `audit` to investigate any failures
4. Generate a summary report

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

