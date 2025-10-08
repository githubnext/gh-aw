---
title: MCP Server
description: Learn how to use the gh-aw MCP server to expose CLI tools via Model Context Protocol, enabling secure integration with AI agents and other MCP clients.
---

The `mcp-server` command implements a Model Context Protocol (MCP) server that exposes gh-aw CLI commands as callable tools. This enables AI agents and other MCP clients to manage agentic workflows programmatically while maintaining security isolation.

## What is MCP?

Model Context Protocol (MCP) is a standardized protocol for connecting AI systems to external tools and data sources. The gh-aw MCP server acts as a bridge between AI agents and the gh-aw CLI, allowing agents to:

- Check workflow status
- Compile workflows to GitHub Actions
- Download and analyze workflow logs
- Investigate workflow runs

## Security Model

The MCP server uses a **subprocess wrapper architecture** where each tool invocation spawns a `gh aw` CLI subprocess. This design ensures that:

- **Token Isolation**: GitHub tokens and other secrets are only accessible to the spawned CLI processes, never to the MCP server process itself
- **Process Separation**: The MCP server process and the agentic workflow process remain separate, preventing credential leakage
- **Fixed Output Directories**: Logs and audit outputs are forced to `/tmp/aw-mcp/logs` to prevent directory traversal attacks

This architecture is critical when exposing CLI functionality to external AI agents.

## Available Tools

The MCP server exposes four core gh-aw commands as tools:

### status

Show workflow file status with optional pattern filtering.

**Parameters:**
- `pattern` (optional): Filter workflows by name pattern

**Example:**
```json
{
  "pattern": "ci-*"
}
```

### compile

Compile markdown workflows to YAML with validation always enabled.

**Parameters:**
- `workflows` (optional): Workflow names to compile (comma-separated)
- `watch` (optional): Watch for changes and recompile automatically
- `purge` (optional): Remove orphaned .lock.yml files
- `no_emit` (optional): Validate without emitting .lock.yml files
- `workflows_dir` (optional): Custom workflows directory

**Example:**
```json
{
  "workflows": "ci-doctor,issue-triage"
}
```

### logs

Download and analyze workflow logs with extensive filtering options. Output is forced to `/tmp/aw-mcp/logs`.

**Parameters:**
- `workflow` (optional): Workflow name to download logs for
- `count` (optional): Number of runs to download
- `start_date` (optional): Filter runs after this date (ISO 8601 or delta time like "-1w")
- `end_date` (optional): Filter runs before this date
- `engine` (optional): Filter by AI engine (claude, codex, copilot)
- `branch` (optional): Filter by branch name
- `after_run_id` (optional): Filter runs after this run ID
- `before_run_id` (optional): Filter runs before this run ID

**Example:**
```json
{
  "workflow": "ci-doctor",
  "count": 10,
  "engine": "claude",
  "start_date": "-1w"
}
```

### audit

Investigate workflow runs and generate detailed reports. Output is forced to `/tmp/aw-mcp/logs`.

**Parameters:**
- `run_id`: Workflow run ID to audit

**Example:**
```json
{
  "run_id": "12345678"
}
```

## Usage

### Starting the Server

**Stdio Transport (Default):**
```bash
gh aw mcp-server
```

The server communicates over stdin/stdout using JSON-RPC.

**HTTP/SSE Transport:**
```bash
gh aw mcp-server --port 3000
```

The server runs an HTTP server on the specified port using Server-Sent Events (SSE) for real-time communication.

### Connecting from MCP Clients

**Stdio Transport:**
```go
import "github.com/modelcontextprotocol/go-sdk/mcp"

client := mcp.NewClient(&mcp.Implementation{
    Name:    "my-client",
    Version: "1.0.0",
}, nil)

transport := &mcp.CommandTransport{
    Command: exec.Command("gh", "aw", "mcp-server"),
}

session, err := client.Connect(ctx, transport, nil)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// List available tools
tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})

// Call a tool
result, err := session.CallTool(ctx, &mcp.CallToolParams{
    Name: "status",
    Arguments: map[string]any{
        "pattern": "ci-*",
    },
})
```

**HTTP Transport:**
```go
client := mcp.NewClient(&mcp.Implementation{
    Name:    "my-client",
    Version: "1.0.0",
}, nil)

transport := &mcp.SSETransport{
    URL: "http://localhost:3000",
}

session, err := client.Connect(ctx, transport, nil)
// Use session as shown above
```

## Integration with Agentic Workflows

The gh-aw MCP server can be integrated into agentic workflows to enable self-management capabilities. A shared workflow configuration is available for easy integration.

### Using the Shared Configuration

Import the `shared/gh-aw-mcp.md` file in your workflow frontmatter:

```aw
---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/gh-aw-mcp.md
---

# Workflow Self-Management Agent

You have access to gh-aw MCP tools to manage workflows in this repository.

Use the `status` tool to check workflow status, `compile` to validate workflows,
`logs` to analyze recent executions, and `audit` to investigate specific runs.

All logs are downloaded to `/tmp/aw-mcp/logs` for analysis.
```

The shared configuration automatically provides:
- MCP server configuration with GITHUB_TOKEN access (HTTP transport)
- Go setup step
- Dependency installation (`make deps-dev`)
- Build step for gh-aw CLI
- Server launch step (starts HTTP server on port 3000)

### Manual Configuration

If you need custom configuration, you can manually define the MCP server:

```yaml
---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  
  - name: Install dependencies
    run: make deps-dev
  
  - name: Build gh-aw CLI
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

# Your workflow content here
```

### Example: Workflow Audit Agent

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

Perform a weekly audit of all agentic workflows in this repository:

1. Use the `status` tool to list all workflows
2. For each workflow:
   - Use the `logs` tool to get the last 5 runs
   - Analyze success rates and failure patterns
   - Use the `audit` tool to investigate any failures
3. Generate a summary report with findings and recommendations

All logs are available in `/tmp/aw-mcp/logs` for detailed analysis.
```

## Best Practices

### Security

- **Never share the MCP server process with untrusted agents**: The server has access to GitHub tokens via environment variables
- **Use HTTP transport in workflows**: Stdio transport is for local CLI usage; workflows should use HTTP/SSE
- **Review output locations**: All logs and audit reports go to `/tmp/aw-mcp/logs`
- **Limit permissions**: Only grant the minimum GitHub token permissions needed (typically `contents: read` and `actions: read`)

### Performance

- **Use filtering options**: When downloading logs, use date filters, run ID filters, or count limits to reduce data transfer
- **Cache builds**: The shared workflow configuration uses Go module caching to speed up builds
- **Background server**: The server launch step starts the MCP server in the background to avoid blocking workflow execution

### Debugging

- **Check server status**: The server logs to stderr, which is visible in GitHub Actions logs
- **Verify tool availability**: Use the MCP client's `ListTools` to confirm all tools are available
- **Test locally**: Run `gh aw mcp-server` locally to test MCP integration before deploying to workflows
- **Examine output files**: Check `/tmp/aw-mcp/logs` for downloaded logs and audit reports

## Troubleshooting

### Server Won't Start

**Problem**: MCP server fails to start in workflow

**Solutions**:
- Ensure Go is set up correctly with `actions/setup-go@v5`
- Verify dependencies are installed with `make deps-dev`
- Check that the build step completes successfully
- Review server logs in GitHub Actions output

### Tools Not Available

**Problem**: MCP client can't find tools

**Solutions**:
- Verify the server is running (check for startup messages in logs)
- Ensure the MCP server configuration includes the correct URL/command
- Check that GITHUB_TOKEN is passed to the server environment

### Token/Permission Errors

**Problem**: Tools fail with authentication errors

**Solutions**:
- Verify `GITHUB_TOKEN` is passed to the MCP server in `env:` section
- Check workflow permissions include `contents: read` and `actions: read`
- Ensure the token has access to the repository and workflow runs

### Output Files Not Found

**Problem**: Can't find downloaded logs or audit reports

**Solutions**:
- All outputs go to `/tmp/aw-mcp/logs` by design
- Check this directory for files
- The directory is created automatically if it doesn't exist
- Files are only created after successful tool execution

## Related Documentation

- [MCP Integration Guide](/gh-aw/guides/mcps/) - Using MCP servers in workflows
- [CLI Commands](/gh-aw/tools/cli/) - Complete CLI reference
- [Security Best Practices](/gh-aw/guides/security/) - Securing agentic workflows
