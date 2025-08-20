# 🔌 Model Context Protocol (MCP) Integration Guide

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
    mcp:
      type: stdio
      command: "python"
      args: ["-m", "trello_mcp"]
      env:
        TRELLO_TOKEN: "${secrets.TRELLO_TOKEN}"
    allowed: ["list_boards"]
---

# Your workflow content here
```

> [!TIP]
> You can inspect test your MCP configuration by running <br/>
> `gh aw inspect <workflow-file>`


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
    mcp:
      type: stdio
      command: "python"
      args: ["-m", "my_mcp_server"]
      env:
        API_KEY: "${secrets.MY_API_KEY}"
        DEBUG: "false"
    allowed: ["process_data", "generate_report"]
```

**Use cases**: Python modules, Node.js scripts, local executables

### 2. Docker Container MCP Servers

Containerized MCP servers for isolation and portability:

```yaml
tools:
  notion:
    mcp:
      type: stdio
      container: "mcp/notion"
      env:
        NOTION_TOKEN: "${secrets.NOTION_TOKEN}"
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
    mcp:
      type: http
      url: "https://api.example.com/mcp"
      headers:
        Authorization: "Bearer ${secrets.API_TOKEN}"
        Content-Type: "application/json"
    allowed: ["query_data", "update_records"]
```

**Use cases**: Cloud services, remote APIs, shared infrastructure

### 4. JSON String Format

Alternative format for complex configurations:

```yaml
tools:
  complex-server:
    mcp: |
      {
        "type": "stdio",
        "command": "python",
        "args": ["-m", "complex_mcp"],
        "env": {
          "API_KEY": "${secrets.API_KEY}",
          "DEBUG": "true"
        }
      }
    allowed: ["process_data", "generate_report"]
```

## GitHub MCP Integration

GitHub Agentic Workflows includes built-in GitHub MCP integration with comprehensive repository access. See [Tools Configuration](tools.md) for details.

You can configure the docker image version for GitHub tools:

```yaml
tools:
  github:
    docker_image_version: "sha-45e90ae"  # Optional: specify version
```

**Configuration Options**:
- `docker_image_version`: Docker image version (default: `"sha-45e90ae"`)

## Tool Allow-listing

When using an agentic engine that allows tool whitelisting (e.g. Claude), you can control which MCP tools are available to your workflow.

### Specific Tools

```yaml
tools:
  custom-server:
    mcp:
      type: stdio
      command: "python"
      args: ["-m", "my_server"]
    allowed: ["tool1", "tool2", "tool3"]
```

When using an agentic engine that allows tool whitelisting (e.g. Claude), this generates tool names: `mcp__servername__tool1`, `mcp__servername__tool2`, etc.

> [!TIP]
> You can inspect the tools available for an Agentic Workflow by running <br/>
> `gh aw inspect <workflow-file>`

### Wildcard Access

```yaml
tools:
  custom-server:
    mcp:
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
    mcp:
      type: http
      url: "https://api.service.com"
      headers:
        Authorization: "Bearer ${secrets.API_TOKEN}"
        X-Custom-Key: "${secrets.CUSTOM_KEY}"
```

## Debugging and Troubleshooting

### MCP Server Inspection

Use the `inspect` command to analyze and troubleshoot MCP configurations:

```bash
# List all workflows with MCP configurations
gh aw inspect

# Inspect all MCP servers in a specific workflow
gh aw inspect my-workflow

# Filter to specific servers
gh aw inspect my-workflow --server trello-server

# Verbose output with connection details
gh aw inspect my-workflow --verbose

# Launch official MCP inspector web interface
gh aw inspect my-workflow --inspector
```

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
2. Check tool name spelling (use `gh aw inspect` to see available tools)
3. Verify MCP server is running correctly

## Related Documentation

- [Tools Configuration](tools.md) - Complete tools reference
- [Commands](commands.md) - CLI commands including `inspect`
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Workflow Structure](workflow-structure.md) - Directory organization

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)