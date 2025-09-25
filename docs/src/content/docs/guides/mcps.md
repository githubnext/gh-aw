---
title: MCP Integration
description: Learn how to use Model Context Protocol (MCP) servers with GitHub Agentic Workflows to connect AI agents to external tools, databases, and services.
---

This guide covers using Model Context Protocol (MCP) servers with GitHub Agentic Workflows.

## What is MCP?

Model Context Protocol (MCP) is a standardized protocol that allows AI agents to connect to external tools, databases, and services in a secure and consistent way. GitHub Agentic Workflows leverages MCP to:

- **Connect to external services**: Integrate with databases, APIs, and third-party tools
- **Extend AI capabilities**: Give your workflows access to specialized functionality
- **Maintain security**: Use standardized authentication and permission controls
- **Enable composability**: Mix and match different MCP servers for complex workflows

## Quick Start

### Basic MCP Configuration

Add MCP servers to your workflow's frontmatter:

```yaml
---
tools:
  github:
    allowed: [get_issue, add_issue_comment]
  
  trello:
    type: stdio
    command: "python"
    args: ["-m", "trello_mcp"]
    env:
      TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["list_boards"]
---

# Your workflow content here
```

> [!TIP]
> You can inspect test your MCP configuration by running <br/>
> `gh aw mcp inspect <workflow-file>`


### Engine Compatibility

Different AI engines support different MCP features:

- **Claude** (default): ✅ Full MCP support (stdio, Docker, HTTP)
- **Codex** (experimental): ✅ Limited MCP support (stdio only, no HTTP)

**Note**: When using Codex engine, HTTP MCP servers will be ignored and only stdio-based servers will be configured.

## MCP Server Types

### 1. Stdio MCP Servers

Direct command execution with stdin/stdout communication:

```yaml
tools:
  python-service:
    type: stdio
    command: "python"
    args: ["-m", "my_mcp_server"]
    env:
      API_KEY: "${{ secrets.MY_API_KEY }}"
      DEBUG: "false"
    allowed: ["process_data", "generate_report"]
```

**Use cases**: Python modules, Node.js scripts, local executables

### 2. Docker Container MCP Servers

Containerized MCP servers for isolation and portability:

```yaml
tools:
  notion:
    type: stdio
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed: ["create_page", "search_pages"]
```

The `container` field automatically generates:
- **Command**: `"docker"`
- **Args**: `["run", "--rm", "-i", "-e", "NOTION_TOKEN", "mcp/notion"]`

**Use cases**: Third-party MCP servers, complex dependencies, security isolation

### 3. HTTP MCP Servers

Remote MCP servers accessible via HTTP (Claude engine only):

```yaml
tools:
  remote-api:
    type: http
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
      Content-Type: "application/json"
    allowed: ["query_data", "update_records"]
```

**Use cases**: Cloud services, remote APIs, shared infrastructure

## GitHub MCP Integration

GitHub Agentic Workflows includes built-in GitHub MCP integration with comprehensive repository access. See [Tools Configuration](/gh-aw/reference/tools/) for details.

You can configure the docker image version for GitHub tools:

```yaml
tools:
  github:
    docker_image_version: "sha-09deac4"  # Optional: specify version
```

**Configuration Options**:
- `docker_image_version`: Docker image version (default: `"sha-09deac4"`)

## Tool Allow-listing

When using an agentic engine that allows tool whitelisting (e.g. Claude), you can control which MCP tools are available to your workflow.

### Specific Tools

```yaml
tools:
  custom-server:
    type: stdio
    command: "python"
    args: ["-m", "my_server"]
    allowed: ["tool1", "tool2", "tool3"]
```

When using an agentic engine that allows tool whitelisting (e.g. Claude), this generates tool names: `mcp__servername__tool1`, `mcp__servername__tool2`, etc.

> [!TIP]
> You can inspect the tools available for an Agentic Workflow by running <br/>
> `gh aw mcp inspect <workflow-file>`

### Wildcard Access

```yaml
tools:
  custom-server:
    type: stdio
    command: "python"
    args: ["-m", "my_server"]
    allowed: ["*"]  # Allow ALL tools from this server
```

When using an agentic engine that allows tool whitelisting (e.g. Claude), this generates: `mcp__servername` (access to all server tools)

### HTTP Headers

```yaml
tools:
  remote-api:
    type: http
    url: "https://api.service.com"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
      X-Custom-Key: "${{ secrets.CUSTOM_KEY }}"
```

## Network Egress Permissions

Restrict outbound network access for containerized MCP servers using a per‑tool domain allowlist. Define allowed domains under `permissions.network.allowed`.

```yaml
tools:
  fetch:
    type: stdio
    container: mcp/fetch
    permissions:
      network:
        allowed:
          - "example.com"
    allowed: ["fetch"]
```

Enforcement in compiled workflows:

- A [Squid proxy](https://www.squid-cache.org/) is generated and pinned to a dedicated Docker network for each proxy‑enabled MCP server.
- The MCP container is configured with `HTTP_PROXY`/`HTTPS_PROXY` to point at Squid; iptables rules only allow egress to the proxy.
- The proxy is seeded with an `allowed_domains.txt` built from your `allowed` list; requests to other domains are blocked.

Notes:

- **Only applies to stdio MCP servers with `container`** - Non‑container stdio and `type: http` servers will cause compilation errors
- Use bare domains without scheme; list each domain you intend to permit.

### Validation Rules

The compiler enforces these network permission rules:

- ❌ **HTTP servers**: `network egress permissions do not apply to remote 'type: http' servers`
- ❌ **Non-container stdio**: `network egress permissions only apply to stdio MCP servers that specify a 'container'`  
- ✅ **Container stdio**: Network permissions work correctly

## MCP Server Management with CLI

GitHub Agentic Workflows provides comprehensive CLI tools for managing MCP servers in your workflows.

### Discovering and Listing MCP Servers

List MCP servers across all workflows or within specific workflows:

```bash
# List all workflows that contain MCP server configurations
gh aw mcp list

# List with detailed MCP server information
gh aw mcp list --verbose

# List MCP servers in a specific workflow
gh aw mcp list weekly-research

# List with detailed configuration
gh aw mcp list weekly-research --verbose
```

The `mcp list` command shows:
- Workflow names containing MCP servers
- Server count per workflow
- In verbose mode: server names, types, commands/URLs, and allowed tools

### Adding MCP Servers from Registry

Add MCP servers to workflows directly from the GitHub MCP registry:

```bash
# Show available MCP servers from registry
gh aw mcp add

# Add an MCP server to a workflow
gh aw mcp add weekly-research makenotion/notion-mcp-server

# Add with specific transport preference (stdio/docker)
gh aw mcp add weekly-research makenotion/notion-mcp-server --transport stdio

# Add with a custom tool ID
gh aw mcp add weekly-research makenotion/notion-mcp-server --tool-id my-notion

# Use a custom MCP registry
gh aw mcp add weekly-research server-name --registry https://custom.registry.com/v1
```

**Registry Features:**
- **Default Registry**: Uses GitHub's MCP registry at `https://api.mcp.github.com/v0`
- **Server Search**: Fuzzy matching by server name
- **Transport Selection**: Automatically chooses best available transport (stdio preferred)
- **Automatic Compilation**: Compiles workflow after adding MCP server

### Exposing gh-aw as MCP Server

Launch an MCP server that exposes gh-aw CLI functionality to AI assistants:

```bash
# Launch MCP server exposing all gh-aw tools
gh aw mcp serve

# Launch with verbose logging
gh aw mcp serve --verbose

# Launch with only specific tools enabled
gh aw mcp serve --allowed-tools compile,logs,run
```

**Available MCP Tools:**
- `compile` - Compile markdown workflow files to YAML
- `logs` - Download and analyze agentic workflow logs
- `mcp_inspect` - Inspect MCP servers and tools
- `mcp_list` - List MCP server configurations
- `mcp_add` - Add MCP tools to workflows
- `run` - Execute workflows on GitHub Actions
- `enable` - Enable workflow execution
- `disable` - Disable workflow execution
- `status` - Show workflow status

This enables AI assistants to manage GitHub Agentic Workflows through the standardized MCP protocol.

## Debugging and Troubleshooting

### MCP Server Inspection

Use the `mcp inspect` command to analyze and troubleshoot MCP configurations:

```bash
# List all workflows with MCP servers configured
gh aw mcp inspect

# Inspect all MCP servers in a specific workflow
gh aw mcp inspect my-workflow

# Inspect a specific MCP server in a workflow
gh aw mcp inspect my-workflow --server trello-server

# Enable verbose output for debugging connection issues
gh aw mcp inspect my-workflow --verbose

# Launch official MCP inspector web interface
gh aw mcp inspect my-workflow --inspector

### Common Issues and Solutions

#### Connection Failures

**Problem**: MCP server fails to connect
```
Error: Failed to connect to MCP server
```

**Solutions**:
1. Check server configuration syntax
2. Verify environment variables are set
3. Test server independently
4. Check network connectivity (for HTTP servers)

#### Permission Denied

**Problem**: Tools not available to workflow
```
Error: Tool 'my_tool' not found
```

**Solutions**:
1. Add tool to `allowed` list
2. Check tool name spelling (use `gh aw mcp inspect` to see available tools)
3. Verify MCP server is running correctly

## Related Documentation

- [Tools Configuration](/gh-aw/reference/tools/) - Complete tools reference
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands including `mcp inspect`
- [Include Directives](/gh-aw/reference/include-directives/) - Modularizing workflows with includes
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory organization

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)
