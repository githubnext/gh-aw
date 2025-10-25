---
tools:
  agentic-workflows:
mcp-servers:
  gh-aw:
    type: http
    url: http://localhost:8765
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  - name: Install dependencies
    run: make deps-dev
  - name: Install binary as 'gh-aw'
    run: make build
  - name: Start MCP server
    run: |
      set -e
      ./gh-aw mcp-server --cmd ./gh-aw --port 8765 &
      MCP_PID=$!
      
      # Wait a moment for server to start
      sleep 2
      
      # Check if server is still running
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "MCP server failed to start"
        exit 1
      fi
      
      echo "MCP server started successfully with PID $MCP_PID"
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---

This shared configuration enables the **agentic-workflows** MCP tool for workflow introspection and analysis.

## Features

The agentic-workflows tool provides:

- **status**: Show status of workflow files in the repository
- **compile**: Compile markdown workflows to YAML
- **logs**: Download and analyze workflow run logs
- **audit**: Investigate workflow run failures and generate reports

## Usage

Import this shared configuration in your workflow:

```yaml
imports:
  - shared/mcp/gh-aw.md
```

## Pre-downloading Logs

For better performance, you can pre-download logs before the AI agent runs by also importing:

```yaml
imports:
  - shared/mcp/gh-aw.md
  - shared/predownload-logs.md
```

This will download logs to `/tmp/gh-aw/aw-mcp/logs` before the AI agent starts, making them
immediately available for analysis without waiting for downloads during execution.

## How It Works

This configuration:
1. Builds the gh-aw CLI tool from source
2. Starts an HTTP MCP server on localhost:8765
3. Configures the agentic-workflows tool to use this server
4. Provides AI agents with tools to analyze workflow execution history
