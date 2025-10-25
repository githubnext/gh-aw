---
title: Using MCPs
description: Learn how to use Model Context Protocol (MCP) servers with GitHub Agentic Workflows to connect AI agents to external tools, databases, and services.
sidebar:
  order: 200
---

This guide covers using Model Context Protocol (MCP) servers with GitHub Agentic Workflows.

## What is MCP?

Model Context Protocol (MCP) is a standardized protocol that allows AI agents to securely connect to external tools, databases, and services. GitHub Agentic Workflows uses MCP to integrate databases and APIs, extend AI capabilities with specialized functionality, maintain standardized security controls, and enable composable workflows by mixing multiple MCP servers.

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
  microsoftdocs:
    url: "https://learn.microsoft.com/api/mcp"
    allowed: ["*"]
  
  notion:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed:
      - "search_pages"
      - "get_page"
      - "get_database"
      - "query_database"
---

# Your workflow content here
```

> [!TIP]
> Inspect MCP configuration: `gh aw mcp inspect <workflow-file>` or find workflows: `gh aw mcp list-tools <server-name>`


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

This automatically searches the registry (default: `https://api.mcp.github.com/v0`), adds server configuration, and compiles the workflow.

### Engine Compatibility

All AI engines support the full range of MCP features:

- **Copilot** (default): ✅ Full MCP support (stdio, Docker, HTTP)
- **Claude**: ✅ Full MCP support (stdio, Docker, HTTP)
- **Codex** (experimental): ✅ Full MCP support (stdio, Docker, HTTP)

## MCP Server Types

### 1. Stdio MCP Servers

Direct command execution with stdin/stdout communication for Python modules, Node.js scripts, and local executables:

```yaml
mcp-servers:
  serena:
    command: "uvx"
    args:
      - "--from"
      - "git+https://github.com/oraios/serena"
      - "serena"
      - "start-mcp-server"
      - "--context"
      - "codex"
      - "--project"
      - "${{ github.workspace }}"
    allowed: ["*"]
```

**npm-based MCP Servers:**

For npm packages used via `npx`, you can automatically track dependencies with Dependabot:

```yaml
mcp-servers:
  playwright:
    command: "npx"
    args: ["-y", "@playwright/mcp@latest"]
    allowed: ["*"]
```

Use `gh aw compile --dependabot` to generate `package.json`, `package-lock.json`, and Dependabot configuration automatically. See [Managing npm Dependencies with Dependabot](/gh-aw/guides/dependabot-npm/).

### 2. Docker Container MCP Servers

Containerized MCP servers for third-party tools, complex dependencies, and security isolation:

```yaml
mcp-servers:
  ast-grep:
    container: "mcp/ast-grep"
    version: "latest"
    allowed: ["*"]
```

The `container` field generates command `"docker"` with args `["run", "--rm", "-i", "mcp/ast-grep:latest"]`. Specify version separately or include it in the container field (e.g., `container: "mcp/ast-grep:latest"`).

#### Container Arguments and Entrypoint Arguments

Specify arguments before (`args`) or after (`entrypointArgs`) the container image following the pattern: `docker run [args] image:tag [entrypointArgs]`

**Example with volume mounts and application arguments**:

```yaml
mcp-servers:
  custom-tool:
    container: "mcp/custom-tool"
    version: "v1.0"
    args:
      - "-v"
      - "/host/data:/app/data"
    entrypointArgs:
      - "serve"
      - "--port"
      - "8080"
      - "--verbose"
    allowed: ["*"]
```

This generates:
```
docker run --rm -i -v /host/data:/app/data mcp/custom-tool:v1.0 serve --port 8080 --verbose
```

**Example with read-only mode** (like the Azure MCP Server):

```yaml
mcp-servers:
  azure:
    container: "mcr.microsoft.com/azure-sdk/azure-mcp"
    version: "latest"
    entrypointArgs:
      - "server"
      - "start"
      - "--read-only"
    env:
      AZURE_TENANT_ID: "${{ secrets.AZURE_TENANT_ID }}"
      AZURE_CLIENT_ID: "${{ secrets.AZURE_CLIENT_ID }}"
      AZURE_CLIENT_SECRET: "${{ secrets.AZURE_CLIENT_SECRET }}"
    allowed: ["*"]
```

This generates:
```
docker run --rm -i -e AZURE_TENANT_ID -e AZURE_CLIENT_ID -e AZURE_CLIENT_SECRET \
  mcr.microsoft.com/azure-sdk/azure-mcp:latest server start --read-only
```

#### Custom Docker Configuration

For advanced use cases, you can configure Docker containers with environment variables and network restrictions:

```yaml
mcp-servers:
  context7:
    container: "mcp/context7"
    env:
      CONTEXT7_API_KEY: "${{ secrets.CONTEXT7_API_KEY }}"
    network:
      allowed:
        - mcp.context7.com
    allowed:
      - get-library-docs
      - resolve-library-id
```

This generates a Docker container with environment variables and network egress controls:
- **Command**: `"docker"`
- **Args**: Includes environment variable flags (e.g., `-e CONTEXT7_API_KEY`) and network proxy configuration
- **Network**: Squid proxy restricts egress to allowed domains only

Configure environment variables (`env:`), Docker arguments (`args:`), application arguments (`entrypointArgs:`), network egress controls (`network.allowed:`), and accessible tools (`allowed:`). See [Container Arguments and Entrypoint Arguments](#container-arguments-and-entrypoint-arguments) and [Network Egress Permissions](#network-egress-permissions).

### 3. HTTP MCP Servers

Remote MCP servers accessible via HTTP for cloud services, remote APIs, and shared infrastructure:

```yaml
mcp-servers:
  microsoftdocs:
    url: "https://learn.microsoft.com/api/mcp"
    allowed: ["*"]

  deepwiki:
    url: "https://mcp.deepwiki.com/sse"
    allowed:
      - read_wiki_structure
      - read_wiki_contents
      - ask_question
```

### 4. Registry-based MCP Servers

Reference MCP servers from the GitHub MCP registry (the `registry` field provides metadata for tooling):

```yaml
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    container: "ghcr.io/microsoft/markitdown"
    allowed: ["*"]
```

## GitHub MCP Integration

GitHub Agentic Workflows includes built-in GitHub MCP integration with comprehensive repository access. See [Tools](/gh-aw/reference/tools/) for details.

Configure the Docker image version (default: `"sha-09deac4"`):

```yaml
tools:
  github:
    version: "sha-09deac4"
```

### GitHub Authentication

Token precedence: `GH_AW_GITHUB_TOKEN` (highest priority) or `GITHUB_TOKEN` (fallback).

## Tool Allow-listing

Control which MCP tools are available with `allowed:` (Claude engine support):

**Specific tools**:
```yaml
mcp-servers:
  deepwiki:
    url: "https://mcp.deepwiki.com/sse"
    allowed:
      - read_wiki_structure
      - read_wiki_contents
```
Generates: `mcp__deepwiki__read_wiki_structure`, `mcp__deepwiki__read_wiki_contents`

**Wildcard access**:
```yaml
mcp-servers:
  ast-grep:
    container: "mcp/ast-grep"
    allowed: ["*"]
```
Generates: `mcp__ast-grep` (all tools)

### HTTP Headers

Configure authentication in URL parameters (e.g., `?apiKey=${{ secrets.API_KEY }}`) or use dedicated header fields when needed.

## Network Egress Permissions

Restrict outbound access for containerized stdio MCP servers with `network.allowed`:

```yaml
mcp-servers:
  context7:
    container: "mcp/context7"
    env:
      CONTEXT7_API_KEY: "${{ secrets.CONTEXT7_API_KEY }}"
    network:
      allowed:
        - mcp.context7.com
    allowed:
      - get-library-docs
```

Enforcement uses a [Squid proxy](https://www.squid-cache.org/) configured with `HTTP_PROXY`/`HTTPS_PROXY` and iptables rules to block non-allowed domains. Only applies to containerized stdio servers (not HTTP or non-container stdio).

## Available Shared MCP Configurations

The gh-aw repository includes pre-configured shared MCP server workflows in `.github/workflows/shared/mcp/`:

### Jupyter Notebook Integration

Execute code in Jupyter notebooks and visualize data:

```yaml
imports:
  - shared/mcp/jupyter.md
```

Provides tools for executing cells, managing notebooks, and retrieving outputs. Includes self-hosted JupyterLab service with Docker services support.

### Drain3 Log Analysis

Mine log patterns and extract structured templates from unstructured log files:

```yaml
imports:
  - shared/mcp/drain3.md
```

Provides 8 tools including `index_file`, `list_clusters`, `find_anomalies`, `compare_runs`, and `search_pattern` for log template mining and pattern analysis.

### Other Available Servers

Additional shared MCP configurations include: AST-Grep, Azure, Brave Search, Context7, DataDog, DeepWiki, Fabric RTI, MarkItDown, Microsoft Docs, Notion, Sentry, Serena, Server Memory, Slack, and Tavily. Browse `.github/workflows/shared/mcp/` for complete list with documentation.

## Debugging and Troubleshooting

Use CLI commands to inspect and debug MCP configurations:

```bash
# Inspect MCP servers in workflow
gh aw mcp inspect my-workflow
gh aw mcp inspect my-workflow --server trello-server --verbose

# List available tools
gh aw mcp list-tools notion my-workflow

# Launch MCP inspector
gh aw mcp inspect my-workflow --inspector
```

For MCP server troubleshooting, import the mcp-debug shared workflow:

```yaml
imports:
  - shared/mcp-debug.md
```

This provides diagnostic tools and the `report_diagnostics_to_pull_request` custom safe-output for posting diagnostic findings to pull requests.

**Common issues**:
- **Connection failures**: Verify configuration syntax, environment variables, and network connectivity
- **Tool not found**: Add tool to `allowed` list and verify spelling with `gh aw mcp inspect`

## Related Documentation

- [Tools](/gh-aw/reference/tools/) - Complete tools reference
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands including `mcp inspect`
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory organization

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)
