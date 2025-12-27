---
title: Web Search
description: How to add web search capabilities to GitHub Agentic Workflows using Tavily MCP server.
sidebar:
  order: 15
---

This guide shows how to add web search to workflows using the Tavily Model Context Protocol (MCP) server, an AI-optimized search provider designed for LLM applications. While alternatives exist (Exa, SerpAPI, Brave Search), this guide focuses on Tavily configuration.

## Tavily Search

[Tavily](https://tavily.com/) provides AI-optimized search with structured JSON responses, news search capability, and fast response times through the [@tavily/mcp-server](https://github.com/tavily-ai/tavily-mcp-server) MCP server.

```aw wrap
---
on: issues
engine: copilot
mcp-servers:
  tavily:
    command: npx
    args: ["-y", "@tavily/mcp-server"]
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed: ["search", "search_news"]
---

# Search and Respond

Search the web for information about: ${{ github.event.issue.title }}

Use the tavily search tool to find recent information.
```

**Setup:**
1. Sign up at [tavily.com](https://tavily.com/)
2. Get your API key from the dashboard
3. Add as repository secret: `gh aw secrets set TAVILY_API_KEY --value "<your-api-key>"`

**Terms of Service:** [Tavily Terms](https://tavily.com/terms)

## MCP Server Configuration

Configure the Tavily MCP server with the `allowed` list to restrict tools, store API keys in GitHub Secrets (never commit them), and use the `-y` flag with npx for automatic installation:

```yaml wrap
mcp-servers:
  tavily:
    command: npx
    args: ["-y", "@tavily/mcp-server"]
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed: ["search", "search_news"]
```

Test your configuration with `gh aw mcp inspect <workflow>`.

## Tool Discovery

To see available tools from the Tavily MCP server:

```bash wrap
# Inspect the MCP server in your workflow
gh aw mcp inspect my-workflow --server tavily

# List tools with details
gh aw mcp list-tools tavily my-workflow --verbose
```

## Network Permissions

Agentic engines explicit network permissions for MCP servers:

```yaml wrap
network:
  allowed:
    - defaults
    - "*.tavily.com"
```

## Related Documentation

- [MCP Integration](/gh-aw/guides/mcps/) - Complete MCP server guide
- [Tools](/gh-aw/reference/tools/) - Tool configuration reference
- [AI Engines](/gh-aw/reference/engines/) - Engine capabilities and limitations
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands including `mcp inspect`
- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [Tavily MCP Server](https://github.com/tavily-ai/tavily-mcp-server)
- [Tavily Documentation](https://tavily.com/)

