---
title: Tools
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools, Playwright, and custom MCP servers.
sidebar:
  order: 700
---

[Tools](/gh-aw/reference/glossary/#tools) are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml wrap
tools:
  edit:
  bash: true
```

Some tools are available by default. All tools declared in imported components are merged into the final workflow.

## Edit Tool (`edit:`)

Allows file editing in the GitHub Actions workspace.

```yaml wrap
tools:
  edit:
```

## Bash Tool (`bash:`)

Enables shell command execution in the workspace. Defaults to safe commands (`echo`, `ls`, `pwd`, `cat`, `head`, `tail`, `grep`, `wc`, `sort`, `uniq`, `date`).

```yaml wrap
tools:
  bash:                              # Default safe commands
  bash: []                           # Disable all commands
  bash: ["echo", "ls", "git status"] # Specific commands only
  bash: [":*"]                       # All commands (use with caution)
```

Use wildcards like `git:*` for command families or `:*` for unrestricted access.

## Web Tools

Enable web content fetching and search capabilities:

```yaml wrap
tools:
  web-fetch:   # Fetch web content
  web-search:  # Search the web (engine-dependent)
```

**Note:** Some engines require third-party Model Context Protocol (MCP) servers for web search. See [Using Web Search](/gh-aw/guides/web-search/).

## GitHub Tools (`github:`)

Configure GitHub API operations including toolsets, remote/local modes, and authentication.

See **[GitHub Tools Reference](/gh-aw/reference/github-tools/)** for complete configuration options.

## Playwright Tool (`playwright:`)

Configure Playwright for browser automation and testing:

```yaml wrap
tools:
  playwright:
    version: "1.56.1"  # Optional: defaults to 1.56.1, use "latest" for newest
```

**Domain Access**: Controlled by the top-level [`network:`](/gh-aw/reference/network/) field. Uses ecosystem bundles (`defaults`, `github`, `node`, `python`, etc.). Defaults to `["localhost", "127.0.0.1"]`. Domains auto-include subdomains.

**GitHub Actions Compatibility**: Playwright runs in a Docker container with security flags required for Chromium to function on GitHub Actions runners (`--security-opt seccomp=unconfined` and `--ipc=host`). These flags are automatically configured by gh-aw version 0.41.0 and later.

## Built-in MCP Tools

### Agentic Workflows (`agentic-workflows:`)

Provides workflow introspection, log analysis, and debugging tools. Requires `actions: read` permission:

```yaml wrap
permissions:
  actions: read
tools:
  agentic-workflows:
```

See [MCP Server](/gh-aw/reference/gh-aw-as-mcp-server/) for available operations.

### Cache Memory (`cache-memory:`)

Persistent memory storage across workflow runs for trends and historical data.

```yaml wrap
tools:
  cache-memory:
```

### Repo Memory (`repo-memory:`)

Repository-specific memory storage for maintaining context across executions.

```yaml wrap
tools:
  repo-memory:
```

## Custom MCP Servers (`mcp-servers:`)

Integrate custom Model Context Protocol servers for third-party services:

```yaml wrap
mcp-servers:
  slack:
    command: "npx"
    args: ["-y", "@slack/mcp-server"]
    env:
      SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
    allowed: ["send_message", "get_channel_history"]
```

**Options**: `command` + `args` (process-based), `container` (Docker image), `url` + `headers` (HTTP endpoint), `registry` (MCP registry URI), `env` (environment variables), `allowed` (tool restrictions). See [MCPs Guide](/gh-aw/guides/mcps/) for setup.

### Registry Field

The `registry` field specifies the URI to an MCP server's installation location in an MCP registry. This is useful for documenting the source of an MCP server and can be used by tooling to discover and install MCP servers:

```yaml wrap
mcp-servers:
  markitdown:
    registry: "https://api.mcp.github.com/v0/servers/microsoft/markitdown"
    command: "npx"
    args: ["-y", "@microsoft/markitdown"]
```

**When to use**:

- **Document server source**: Include `registry` to indicate where the MCP server is published
- **Registry-aware tooling**: Some tools may use the registry URI for discovery and version management
- **Both stdio and HTTP servers**: Works with both `command`-based stdio servers and `url`-based HTTP servers

**Examples**:

```yaml wrap
# Stdio server with registry
mcp-servers:
  filesystem:
    registry: "https://api.mcp.github.com/v0/servers/modelcontextprotocol/filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem"]

# HTTP server with registry
mcp-servers:
  custom-api:
    registry: "https://registry.example.com/servers/custom-api"
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
```

The `registry` field is informational and does not affect server execution. It complements other configuration fields like `command`, `args`, `container`, or `url`.

## Related Documentation

- [GitHub Tools](/gh-aw/reference/github-tools/) - GitHub API operations, toolsets, and modes
- [Safe Inputs](/gh-aw/reference/safe-inputs/) - Define custom inline tools with JavaScript or shell scripts
- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
- [Network Permissions](/gh-aw/reference/network/) - Network access control for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Complete Model Context Protocol setup and usage
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
