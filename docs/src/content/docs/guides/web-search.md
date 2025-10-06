---
title: Web Search with MCP
description: Learn how to add web search capabilities to GitHub Agentic Workflows using third-party MCP servers like Tavily, Exa, and SerpAPI.
---

This guide covers how to add web search capabilities to workflows using third-party MCP servers.

## Overview

Some AI engines (like Copilot) don't include built-in web search functionality. To add web search capabilities to these workflows, you can integrate third-party MCP servers that provide search functionality.

This approach works with all AI engines and gives you control over which search provider to use based on your needs.

## Available Search Providers

### Tavily Search

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

### Exa Search

[Exa](https://exa.ai/) is a search engine optimized for AI applications with semantic search capabilities.

**MCP Server:** [@exa-labs/mcp-server-exa](https://github.com/exa-labs/exa-mcp-server)

```aw
---
on: issues
engine: copilot
mcp-servers:
  exa:
    command: npx
    args: ["-y", "@exa-labs/mcp-server-exa"]
    env:
      EXA_API_KEY: "${{ secrets.EXA_API_KEY }}"
    allowed: ["search", "find_similar", "get_contents"]
---

# Research and Summarize

Research the topic: ${{ github.event.issue.title }}

Use the exa search tool to find relevant content and similar pages.
```

**Features:**
- Semantic search with neural ranking
- Find similar pages
- Extract content from search results
- High-quality results for research

**Setup:**
1. Sign up at [exa.ai](https://exa.ai/)
2. Get your API key from settings
3. Add as repository secret: `gh secret set EXA_API_KEY -a actions --body "<your-api-key>"`

**Terms of Service:** [Exa Terms](https://exa.ai/terms)

### SerpAPI

[SerpAPI](https://serpapi.com/) provides access to Google and other search engines with structured data extraction.

**MCP Server:** [@serpapi/mcp-server](https://github.com/serpapi/mcp-server) (community-maintained)

```aw
---
on: issues
engine: copilot
mcp-servers:
  serpapi:
    command: npx
    args: ["-y", "@serpapi/mcp-server"]
    env:
      SERPAPI_API_KEY: "${{ secrets.SERPAPI_API_KEY }}"
    allowed: ["google_search", "google_news", "google_images"]
---

# Search and Report

Search for: ${{ github.event.issue.title }}

Use the serpapi google_search tool to find current information.
```

**Features:**
- Access to Google Search results
- News, images, and shopping search
- Location-based search
- Structured data extraction

**Setup:**
1. Sign up at [serpapi.com](https://serpapi.com/)
2. Get your API key from your account
3. Add as repository secret: `gh secret set SERPAPI_API_KEY -a actions --body "<your-api-key>"`

**Terms of Service:** [SerpAPI Terms](https://serpapi.com/terms)

### Brave Search

[Brave Search](https://brave.com/search/api/) provides privacy-focused web search with no tracking.

**MCP Server:** Custom implementation using `@modelcontextprotocol/server-brave-search`

```aw
---
on: issues
engine: copilot
mcp-servers:
  brave-search:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-brave-search"]
    env:
      BRAVE_API_KEY: "${{ secrets.BRAVE_API_KEY }}"
    allowed: ["brave_web_search", "brave_local_search"]
---

# Privacy-focused Search

Search for information about: ${{ github.event.issue.title }}

Use brave_web_search for privacy-focused results.
```

**Features:**
- Privacy-focused (no tracking)
- Independent search index
- Web and local search
- Clean, ad-free results

**Setup:**
1. Sign up at [brave.com/search/api](https://brave.com/search/api/)
2. Get your API key
3. Add as repository secret: `gh secret set BRAVE_API_KEY -a actions --body "<your-api-key>"`

**Terms of Service:** [Brave Search API Terms](https://brave.com/search/api/terms/)

## Choosing a Search Provider

Consider these factors when selecting a search provider:

**For AI/LLM Applications:**
- **Tavily**: Best for AI-optimized results with structured data
- **Exa**: Best for semantic search and research tasks

**For General Web Search:**
- **SerpAPI**: Best for accessing Google results with full feature parity
- **Brave Search**: Best for privacy-focused applications

**Cost Considerations:**
- All providers offer free tiers for testing
- Check pricing pages for production usage limits
- Monitor your API usage to avoid unexpected costs

## MCP Server Configuration

All search MCP servers follow the same basic pattern:

```yaml
mcp-servers:
  <provider-name>:
    command: npx                           # Use npx for npm packages
    args: ["-y", "<package-name>"]         # -y to auto-install
    env:
      <API_KEY_VAR>: "${{ secrets.<SECRET_NAME> }}"
    allowed: ["<tool1>", "<tool2>"]       # Specific tools to allow
```

**Best Practices:**
1. Always use the `allowed` list to restrict which tools can be used
2. Store API keys in GitHub Secrets, never commit them
3. Use `-y` flag with npx to ensure automatic installation
4. Test MCP configuration with `gh aw mcp inspect <workflow-name>`

## Tool Discovery

To see available tools from a search MCP server:

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
    - "*.exa.ai"            # Exa API
    - "*.serpapi.com"       # SerpAPI
    - "*.brave.com"         # Brave Search API
```

The Copilot engine doesn't require explicit network permissions as MCP servers run with network access by default.

## Troubleshooting

### API Key Issues

**Problem:** MCP server fails with authentication error
```
Error: Invalid API key
```

**Solutions:**
1. Verify the secret is set: `gh secret list`
2. Check the secret name matches the workflow configuration
3. Ensure the API key is valid and active in the provider's dashboard

### Rate Limiting

**Problem:** Search requests fail with rate limit errors
```
Error: Rate limit exceeded
```

**Solutions:**
1. Check your API usage on the provider's dashboard
2. Upgrade to a higher tier if needed
3. Implement caching or reduce search frequency
4. Add error handling in your workflow instructions

### MCP Connection Failures

**Problem:** MCP server fails to start
```
Error: Failed to connect to MCP server
```

**Solutions:**
1. Verify the package name is correct
2. Check that npx can install the package
3. Test the MCP server configuration: `gh aw mcp inspect <workflow-name>`
4. Review workflow logs for detailed error messages

## Related Documentation

- [MCP Integration](/gh-aw/guides/mcps/) - Complete MCP server guide
- [Tools Configuration](/gh-aw/reference/tools/) - Tool configuration reference
- [AI Engines](/gh-aw/reference/engines/) - Engine capabilities and limitations
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands including `mcp inspect`

## External Resources

- [Model Context Protocol Specification](https://github.com/modelcontextprotocol/specification)
- [Tavily MCP Server](https://github.com/tavily-ai/tavily-mcp-server)
- [Exa MCP Server](https://github.com/exa-labs/exa-mcp-server)
- [Brave Search MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search)
