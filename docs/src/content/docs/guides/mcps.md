---
title: Using MCPs
description: How to use Model Context Protocol (MCP) servers with GitHub Agentic Workflows to connect AI agents to external tools, databases, and services.
sidebar:
  order: 4
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

```bash wrap
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

## MCP Server Types

### 1. Stdio MCP Servers

Execute commands with stdin/stdout communication for Python modules, Node.js scripts, and local executables:

```yaml wrap
mcp-servers:
  serena:
    command: "uvx"
    args: ["--from", "git+https://github.com/oraios/serena", "serena"]
    allowed: ["*"]
```

### 2. Docker Container MCP Servers

Run containerized MCP servers with environment variables, volume mounts, and network restrictions:

```yaml wrap
mcp-servers:
  ast-grep:
    container: "mcp/ast-grep:latest"
    allowed: ["*"]

  azure:
    container: "mcr.microsoft.com/azure-sdk/azure-mcp:latest"
    entrypointArgs: ["server", "start", "--read-only"]
    env:
      AZURE_TENANT_ID: "${{ secrets.AZURE_TENANT_ID }}"
    allowed: ["*"]

  custom-tool:
    container: "mcp/custom-tool:v1.0"
    args: ["-v", "/host/data:/app/data"]  # Volume mounts before image
    entrypointArgs: ["serve", "--port", "8080"]  # App args after image
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    network:
      allowed: ["api.example.com"]  # Restricts egress to allowed domains
    allowed: ["tool1", "tool2"]
```

The `container` field generates `docker run --rm -i <args> <image> <entrypointArgs>`. Network restrictions use a Squid proxy and apply only to containerized stdio servers.

### 3. HTTP MCP Servers

Remote MCP servers accessible via HTTP for cloud services, remote APIs, and shared infrastructure:

```yaml wrap
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

```yaml wrap
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    container: "ghcr.io/microsoft/markitdown"
    allowed: ["*"]
```

## GitHub MCP Integration

GitHub Agentic Workflows includes built-in GitHub MCP integration with comprehensive repository access. See [Tools](/gh-aw/reference/tools/) for details.

:::tip[Use Toolsets for GitHub Tools]
Prefer using `toolsets:` instead of `allowed:` for GitHub tools. Toolsets provide better organization and ensure complete functionality. See the [GitHub Toolsets](/gh-aw/reference/tools/#github-toolsets) section for details.
:::

Configure the Docker image version (default: `"sha-09deac4"`):

```yaml wrap
tools:
  github:
    version: "sha-09deac4"
    toolsets: [default, actions]  # Recommended: use toolsets
```

### GitHub Authentication

Token precedence: `GH_AW_GITHUB_TOKEN` (highest priority) or `GITHUB_TOKEN` (fallback).

## Tool Allow-listing

Control tool access with `allowed:` - use specific tool names for granular control or `["*"]` for all tools. Tool names generate as `mcp__<server>__<tool>` (or `mcp__<server>` for wildcards).

**HTTP Headers**: Configure authentication in URL parameters (e.g., `?apiKey=${{ secrets.API_KEY }}`).

## Network Egress Permissions

Restrict outbound access for containerized stdio MCP servers using `network.allowed` (see [Docker Container example](#2-docker-container-mcp-servers)). Enforcement uses a [Squid proxy](https://www.squid-cache.org/) with `HTTP_PROXY`/`HTTPS_PROXY` and iptables rules. Only applies to containerized stdio servers.

## Available Shared MCP Configurations

Pre-configured MCP servers in `.github/workflows/shared/mcp/` can be imported into workflows:

| MCP Server | Import Path | Key Capabilities |
|------------|-------------|------------------|
| **Jupyter** | `shared/mcp/jupyter.md` | Execute code, manage notebooks, visualize data |
| **Drain3** | `shared/mcp/drain3.md` | Log pattern mining with 8 tools including `index_file`, `list_clusters`, `find_anomalies` |
| **Others** | `shared/mcp/*.md` | AST-Grep, Azure, Brave Search, Context7, DataDog, DeepWiki, Fabric RTI, MarkItDown, Microsoft Docs, Notion, Sentry, Serena, Server Memory, Slack, Tavily |

## Debugging and Troubleshooting

Inspect MCP configurations with CLI commands: `gh aw mcp inspect my-workflow` (add `--server <name> --verbose` for details) or `gh aw mcp list-tools <server> my-workflow`.

For advanced debugging, import `shared/mcp-debug.md` to access diagnostic tools and the `report_diagnostics_to_pull_request` custom safe-output.

**Common issues**: Connection failures (verify syntax, env vars, network) or tool not found (check `allowed` list with `gh aw mcp inspect`).

## Related Documentation

- [Tools](/gh-aw/reference/tools/) - Complete tools reference
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands including `mcp inspect`
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory organization

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)
