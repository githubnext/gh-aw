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

Add MCP servers to your workflow's frontmatter using the `mcp-servers:` section:

```aw wrap
---
on: issues
permissions:
  contents: read
  issues: write
engine: claude
mcp-servers:
  trello:
    command: "python"
    args: ["-m", "trello_mcp"]
    env:
      TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["list_boards", "create_card"]
  
  notion:
    registry: https://api.mcp.github.com/v0/servers/makenotion/notion-mcp-server
    command: npx
    args: ["-y", "@makenotion/notion-mcp-server"]
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed: ["search_pages", "create_page"]
---

# Your workflow content here
```

> [!TIP]
> You can inspect and test your MCP configuration by running: <br/>
> `gh aw mcp inspect <workflow-file>` for full inspection <br/>
> `gh aw mcp list-tools <server-name>` to find workflows with the server


### Adding MCP Servers from the Registry

The easiest way to add MCP servers is using the GitHub MCP registry with the `gh aw mcp add` command:

```bash
# List available MCP servers from the registry
gh aw mcp add

# Add a specific MCP server to your workflow
gh aw mcp add my-workflow makenotion/notion-mcp-server

# Add with specific transport preference
gh aw mcp add my-workflow makenotion/notion-mcp-server --transport stdio

# Add with custom tool ID
gh aw mcp add my-workflow makenotion/notion-mcp-server --tool-id my-notion

# Use a custom registry
gh aw mcp add my-workflow server-name --registry https://custom.registry.com/v1
```

This command automatically:
- Searches the MCP registry for the specified server
- Adds the server configuration to your workflow's `mcp-servers:` section (using the modern format)
- Compiles the workflow to generate the `.lock.yml` file

**Default Registry**: `https://api.mcp.github.com/v0`

### Engine Compatibility

Different AI engines support different MCP features:

- **Copilot** (default): ✅ Full MCP support (stdio, Docker, HTTP)
- **Claude**: ✅ Full MCP support (stdio, Docker, HTTP)
- **Codex** (experimental): ✅ Limited MCP support (stdio only, no HTTP)

**Note**: When using Codex engine, HTTP MCP servers will be ignored and only stdio-based servers will be configured.

## MCP Server Types

### 1. Stdio MCP Servers

Direct command execution with stdin/stdout communication:

```yaml
mcp-servers:
  python-service:
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
mcp-servers:
  notion:
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
mcp-servers:
  remote-api:
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
      Content-Type: "application/json"
    allowed: ["query_data", "update_records"]
```

**Use cases**: Cloud services, remote APIs, shared infrastructure

### 4. Registry-based MCP Servers

MCP servers that reference entries in the GitHub MCP registry:

```yaml
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    command: npx
    args: ["-y", "@microsoft/markitdown"]
    allowed: ["convert_html", "convert_pdf"]
```

**Registry Reference**: The `registry` field provides metadata about the MCP server's origin and can help with tooling and documentation.

## GitHub MCP Integration

GitHub Agentic Workflows includes built-in GitHub MCP integration with comprehensive repository access. See [Tools Configuration](/gh-aw/reference/tools/) for details.

You can configure the docker image version for GitHub tools:

```yaml
tools:
  github:
    version: "sha-09deac4"  # Optional: specify version
```

**Configuration Options**:
- `version`: Version (default: `"sha-09deac4"`)

## Tool Allow-listing

When using an agentic engine that supports tool allow-listing (e.g. Claude), you can control which MCP tools are available to your workflow.

### Specific Tools

```yaml
mcp-servers:
  custom-server:
    command: "python"
    args: ["-m", "my_server"]
    allowed: ["tool1", "tool2", "tool3"]
```

When using an agentic engine that supports tool allow-listing (e.g. Claude), this generates tool names: `mcp__servername__tool1`, `mcp__servername__tool2`, etc.

> [!TIP]
> You can inspect tools available from MCP servers by running: <br/>
> `gh aw mcp inspect <workflow-file>` for full inspection <br/>
> `gh aw mcp list-tools <server-name> [workflow-file]` for focused tool listing

### Wildcard Access

```yaml
mcp-servers:
  custom-server:
    command: "python"
    args: ["-m", "my_server"]
    allowed: ["*"]  # Allow ALL tools from this server
```

When using an agentic engine that supports tool allow-listing (e.g. Claude), this generates: `mcp__servername` (access to all server tools)

### HTTP Headers

HTTP headers can be configured for remote MCP servers:

```yaml
mcp-servers:
  remote-api:
    url: "https://api.service.com"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
      X-Custom-Key: "${{ secrets.CUSTOM_KEY }}"
```

## Network Egress Permissions

Restrict outbound network access for containerized MCP servers using a per‑tool domain allowlist. Define allowed domains under `network.allowed`.

```yaml
mcp-servers:
  fetch:
    container: mcp/fetch
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

- ❌ **HTTP servers**: `network egress permissions do not apply to remote HTTP MCP servers`
- ❌ **Non-container stdio**: `network egress permissions only apply to containerized MCP servers`  
- ✅ **Container stdio**: Network permissions work correctly

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
```

### MCP Tool Discovery

Use the `mcp list-tools` command to explore available tools from specific MCP servers:

```bash
# Find workflows containing a specific MCP server
gh aw mcp list-tools github

# List tools available from an MCP server in a specific workflow
gh aw mcp list-tools notion my-workflow

# List tools with detailed descriptions and permission status
gh aw mcp list-tools trello my-workflow --verbose
```

This command is particularly useful for:
- **Exploring capabilities**: See what tools are available from each MCP server
- **Workflow discovery**: Find which workflows use a specific MCP server
- **Permission debugging**: Check which tools are allowed in your workflow configuration

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
