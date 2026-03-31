---
title: Using Custom MCP Servers
description: How to use Model Context Protocol (MCP) servers with GitHub Agentic Workflows to connect AI agents to external tools, databases, and services.
sidebar:
  order: 4
---

[Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP) is a standard for AI tool integration, allowing agents to securely connect to external tools, databases, and services. GitHub Agentic Workflows includes built-in GitHub MCP integration (see [GitHub Tools](/gh-aw/reference/github-tools/)); this guide covers adding custom MCP servers for external services.

> [!IMPORTANT]
>
> Custom MCP servers should be **read-only**. Write operations must go through [safe outputs](/gh-aw/reference/safe-outputs/) or [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/). Ensure your MCP server implements authentication and authorization to prevent unauthorized write access.

## Manually Configuring a Custom MCP Server

Add MCP servers to your workflow's frontmatter using the `mcp-servers:` section:

```aw wrap
---
on: issues

permissions:
  contents: read

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
    - api.example.com
```

The `container` field generates `docker run --rm -i <args> <image> <entrypointArgs>`. 

### HTTP MCP Servers

Remote MCP servers accessible via HTTP. Configure authentication using the `headers` field for static API keys, or the `auth` field for dynamic token acquisition:

```yaml wrap
mcp-servers:
  deepwiki:
    url: "https://mcp.deepwiki.com/sse"
    allowed:
      - read_wiki_structure
      - read_wiki_contents
      - ask_question

  authenticated-api:
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
    allowed: ["*"]
```

#### GitHub Actions OIDC Authentication

For MCP servers that accept GitHub Actions OIDC tokens, use the `auth` field instead of a static `headers` value. The gateway acquires a short-lived JWT from the GitHub Actions OIDC endpoint and injects it as an `Authorization: Bearer` header on every outgoing request.

```yaml wrap
permissions:
  id-token: write   # required for OIDC token acquisition

mcp-servers:
  my-secure-server:
    url: "https://my-server.example.com/mcp"
    auth:
      type: github-oidc
      audience: "https://my-server.example.com"  # optional; defaults to the server URL
    allowed: ["*"]
```

The `auth.type: github-oidc` field is only valid on HTTP servers. The MCP server is responsible for validating the token; the gateway acts as a token forwarder. See [MCP Gateway — Upstream Authentication](/gh-aw/reference/mcp-gateway/#76-upstream-authentication-oidc) for full specification details.

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

Use `allowed:` to specify which tools are available, or `["*"]` to allow all:

```yaml wrap
mcp-servers:
  notion:
    container: "mcp/notion"
    allowed: ["search_pages", "get_page"]  # or ["*"] to allow all
```

## Shared MCP Configurations

Pre-configured MCP server specifications are available in [`.github/workflows/shared/mcp/`](https://github.com/github/gh-aw/tree/main/.github/workflows/shared/mcp) and can be copied or imported directly. Examples include:

| MCP Server | Import Path | Key Capabilities |
|------------|-------------|------------------|
| **Jupyter** | `shared/mcp/jupyter.md` | Execute code, manage notebooks, visualize data |
| **Drain3** | `shared/mcp/drain3.md` | Log pattern mining with 8 tools including `index_file`, `list_clusters`, `find_anomalies` |
| **Others** | `shared/mcp/*.md` | AST-Grep, Azure, Brave Search, Context7, DataDog, DeepWiki, Fabric RTI, MarkItDown, Microsoft Docs, Notion, Sentry, Serena, Server Memory, Slack, Tavily |

## Adding MCP Servers from the Registry

Use `gh aw mcp add` to browse and add servers from the GitHub MCP registry (default: `https://api.mcp.github.com/v0`):

```bash wrap
gh aw mcp add                                                                    # List available servers
gh aw mcp add my-workflow makenotion/notion-mcp-server                           # Add server
gh aw mcp add my-workflow makenotion/notion-mcp-server --transport stdio         # Specify transport
gh aw mcp add my-workflow makenotion/notion-mcp-server --tool-id my-notion       # Custom tool ID
gh aw mcp add my-workflow server-name --registry https://custom.registry.com/v1  # Custom registry
```

## Debugging and Troubleshooting

Inspect MCP configurations with CLI commands: `gh aw mcp inspect my-workflow` (add `--server <name> --verbose` for details) or `gh aw mcp list-tools <server> my-workflow`.

For advanced debugging, import `shared/mcp-debug.md` to access diagnostic tools and the `report_diagnostics_to_pull_request` custom safe-output.

**Common issues**: Connection failures (verify syntax, env vars, network) or tool not found (check toolsets configuration or `allowed` list with `gh aw mcp inspect`).

## Related Documentation

- [MCP Scripts](/gh-aw/reference/mcp-scripts/) - Define custom inline tools without external MCP servers
- [Tools](/gh-aw/reference/tools/) - Complete tools reference
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands including `mcp inspect`
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory organization
- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)
