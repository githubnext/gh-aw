---
title: Web Search
description: How to add web search capabilities to GitHub Agentic Workflows using Tavily MCP server.
sidebar:
  order: 15
---

This guide shows how to add web search to workflows using the Tavily Model Context Protocol (MCP) server, an AI-optimized search provider designed for LLM applications. While alternatives exist (Exa, SerpAPI, Brave Search), this guide focuses on Tavily configuration.

## Tavily Search

[Tavily](https://tavily.com/) provides AI-optimized search with structured JSON responses, news search capability, and fast response times through the [@tavily/mcp](https://github.com/tavily-ai/tavily-mcp) MCP server.

```aw wrap
---
on: issues
engine: copilot
mcp-servers:
  tavily:
    command: npx
    args: ["-y", "@tavily/mcp"]
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed: ["search", "search_news"]
---

# Search and Respond

Search the web for information about: ${{ github.event.issue.title }}

Use the tavily search tool to find recent information.
```

**Setup:**
1. Sign up at [tavily.com](https://tavily.com/) and get your API key
2. Add as repository secret: `gh aw secrets set TAVILY_API_KEY --value "<your-api-key>"`

[Tavily Terms of Service](https://tavily.com/terms)

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

Agentic workflows require explicit network permissions for MCP servers:

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
- [Tavily MCP Server](https://github.com/tavily-ai/tavily-mcp)
- [Tavily Documentation](https://tavily.com/)

