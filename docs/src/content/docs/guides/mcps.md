---
title: Using MCPs
description: How to use Model Context Protocol (MCP) servers with GitHub Agentic Workflows to connect AI agents to external tools, databases, and services.
sidebar:
  order: 4
---

This guide covers using [Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP) servers with GitHub Agentic Workflows.

[Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP, a standard for AI tool integration) is a standardized protocol that allows AI agents to securely connect to external tools, databases, and services. GitHub Agentic Workflows uses MCP to integrate databases and APIs, extend AI capabilities with specialized functionality, maintain standardized security controls, and enable composable workflows by mixing multiple MCP servers.

GitHub Agentic Workflows includes built-in GitHub MCP integration with comprehensive repository access. See [Tools](/gh-aw/reference/tools/) for details. This guide focuses on using custom MCP servers to connect to external services and databases.

## Manually Configuring a Custom MCP Server

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

## Custom MCP Server Types

### Stdio MCP Servers

Execute commands with stdin/stdout communication for Python modules, Node.js scripts, and local executables:

```yaml wrap
mcp-servers:
  serena:
    command: "uvx"
    args: ["--from", "git+https://github.com/oraios/serena", "serena"]
    allowed: ["*"]
```

### Docker Container MCP Servers

Run containerized MCP servers with environment variables, volume mounts, and network restrictions:

```yaml wrap
mcp-servers:
  ast-grep:
    container: "mcp/ast-grep:latest"
    allowed: ["*"]

  custom-tool:
    container: "mcp/custom-tool:v1.0"
    args: ["-v", "/host/data:/app/data"]  # Volume mounts before image
    entrypointArgs: ["serve", "--port", "8080"]  # App args after image
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed: ["tool1", "tool2"]

network:
  allowed:
    - defaults
    - api.example.com  # Restricts egress to allowed domains
```

The `container` field generates `docker run --rm -i <args> <image> <entrypointArgs>`. 

### HTTP MCP Servers

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

HTTP MCP servers must implement the MCP specification over HTTP. Configure authentication headers using the `headers` field:

```yaml wrap
mcp-servers:
  authenticated-api:
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
      X-Custom-Header: "value"
    allowed: ["*"]
```

Headers are injected into all HTTP requests made to the MCP server, enabling bearer token authentication, API keys, and other custom authentication schemes.

### Registry-based MCP Servers

Reference MCP servers from the GitHub MCP registry (the `registry` field provides metadata for tooling):

```yaml wrap
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    container: "ghcr.io/microsoft/markitdown"
    allowed: ["*"]
```

## MCP Tool Filtering

For custom MCP servers, use `allowed:` to specify which tools are available:

```yaml wrap
mcp-servers:
  notion:
    container: "mcp/notion"
    allowed: ["search_pages", "get_page"]  # Limit to specific tools
```

Use `["*"]` to allow all tools from a custom MCP server.

## Shared MCP Configurations

Pre-configured MCP server specifications are available in the GitHub Agentics Workflow repository [`.github/workflows/shared/mcp/`](https://github.com/github/gh-aw/tree/main/.github/workflows/shared/mcp) for common tools and services. These can be copied into your own workflows or imported directly. Examples include:

| MCP Server | Import Path | Key Capabilities |
|------------|-------------|------------------|
| **Jupyter** | `shared/mcp/jupyter.md` | Execute code, manage notebooks, visualize data |
| **Drain3** | `shared/mcp/drain3.md` | Log pattern mining with 8 tools including `index_file`, `list_clusters`, `find_anomalies` |
| **Others** | `shared/mcp/*.md` | AST-Grep, Azure, Brave Search, Context7, DataDog, DeepWiki, Fabric RTI, MarkItDown, Microsoft Docs, Notion, Sentry, Serena, Server Memory, Slack, Tavily |

## Adding MCP Servers from the Registry

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

## Debugging and Troubleshooting

Inspect MCP configurations with CLI commands: `gh aw mcp inspect my-workflow` (add `--server <name> --verbose` for details) or `gh aw mcp list-tools <server> my-workflow`.

For advanced debugging, import `shared/mcp-debug.md` to access diagnostic tools and the `report_diagnostics_to_pull_request` custom safe-output.

**Common issues**: Connection failures (verify syntax, env vars, network) or tool not found (check toolsets configuration or `allowed` list with `gh aw mcp inspect`).

## Related Documentation

- [Safe Inputs](/gh-aw/reference/safe-inputs/) - Define custom inline tools without external MCP servers
- [Tools](/gh-aw/reference/tools/) - Complete tools reference
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands including `mcp inspect`
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory organization

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)
