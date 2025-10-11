---
title: Web Search Guide
description: Learn how to add web search capabilities to GitHub Agentic Workflows using Tavily MCP server.
---

This guide covers how to add web search capabilities to workflows using the Tavily MCP server.

## Overview

Some AI engines (like Copilot) don't include built-in web search functionality. To add web search capabilities to these workflows, you can integrate third-party MCP servers that provide search functionality.

This guide focuses on Tavily, an AI-optimized search provider designed for LLM applications. Other alternatives include Exa (semantic search), SerpAPI (Google search access), and Brave Search (privacy-focused), though this guide only covers Tavily setup.

## Tavily Search

[Tavily](https://tavily.com/) provides AI-optimized search designed for LLM applications with structured results.

**MCP Server:** [@tavily/mcp-server](https://github.com/tavily-ai/tavily-mcp-server)

```aw
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

**Features:**
- AI-optimized search results
- News search capability
- Structured JSON responses
- Fast response times

**Setup:**
1. Sign up at [tavily.com](https://tavily.com/)
2. Get your API key from the dashboard
3. Add as repository secret: `gh secret set TAVILY_API_KEY -a actions --body "<your-api-key>"`

**Terms of Service:** [Tavily Terms](https://tavily.com/terms)

## MCP Server Configuration

Tavily MCP server follows this basic pattern:

```yaml
mcp-servers:
  tavily:
    command: npx                           # Use npx for npm packages
    args: ["-y", "@tavily/mcp-server"]     # -y to auto-install
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed: ["search", "search_news"]    # Specific tools to allow
```

**Best Practices:**
1. Always use the `allowed` list to restrict which tools can be used
2. Store API keys in GitHub Secrets, never commit them
3. Use `-y` flag with npx to ensure automatic installation
4. Test MCP configuration with `gh aw mcp inspect <workflow-name>`

## Tool Discovery

To see available tools from the Tavily MCP server:

```bash
# Inspect the MCP server in your workflow
gh aw mcp inspect my-workflow --server tavily

# List tools with details
gh aw mcp list-tools tavily my-workflow --verbose
```

## Network Permissions

Some engines (like Claude) require explicit network permissions for MCP servers to access external APIs:

```yaml
engine: claude
network:
  allowed:
    - defaults              # Basic infrastructure
    - "*.tavily.com"        # Tavily API
```

The Copilot engine doesn't require explicit network permissions as MCP servers run with network access by default.

## Related Documentation

- [MCP Integration](/gh-aw/guides/mcps/) - Complete MCP server guide
- [Tools Configuration](/gh-aw/reference/tools/) - Tool configuration reference
- [AI Engines](/gh-aw/reference/engines/) - Engine capabilities and limitations
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands including `mcp inspect`

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [Tavily MCP Server](https://github.com/tavily-ai/tavily-mcp-server)
- [Tavily Documentation](https://tavily.com/)

