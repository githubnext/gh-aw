---
mcp-servers:
  dev-proxy:
    command: "npx"
    args:
      - "@devproxy/mcp"
    allowed: ["*"]
steps:
  - name: Install Dev Proxy
    uses: dev-proxy-tools/actions/setup@v1
    with:
      auto-start: false
---

## Dev Proxy MCP Server

Microsoft Dev Proxy is a command-line tool that simulates API behaviors, helping you test your applications for various scenarios including:

- **API Mocking**: Simulate APIs before they're implemented
- **Error Simulation**: Test how your app handles API errors, throttling, and rate limiting
- **Network Conditions**: Simulate slow or unreliable network connections
- **Authentication Testing**: Test different authentication scenarios
- **Response Manipulation**: Modify API responses to test edge cases

### Configuration

This shared workflow runs the Dev Proxy MCP server using npx, providing access to Dev Proxy functionality through the Model Context Protocol.

**Command**: `npx @devproxy/mcp`

**Required Setup**: The workflow automatically installs Dev Proxy using the official GitHub Action from dev-proxy-tools.

### Setup

1. Include this configuration in your workflow:
   ```yaml
   imports:
     - shared/mcp/dev-proxy.md
   ```

2. No additional secrets or environment variables are required for basic usage.

### Example Usage

```aw
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: claude
imports:
  - shared/mcp/dev-proxy.md
---

# API Testing Assistant

Analyze the API testing requirements mentioned in issue #${{ github.event.issue.number }}.

Use the Dev Proxy MCP tools to help create mock API responses and test scenarios for the described API endpoints.
```

### Available Tools

The Dev Proxy MCP server provides tools for:
- Configuring proxy settings and API mocks
- Defining API response behaviors
- Setting up error simulation scenarios
- Managing rate limiting and throttling rules

Specific tools can be discovered using the MCP protocol when the server is running.

### More Information

- **Official Documentation**: https://learn.microsoft.com/en-us/microsoft-cloud/dev/dev-proxy/
- **GitHub Actions Integration**: https://learn.microsoft.com/en-us/microsoft-cloud/dev/dev-proxy/how-to/use-dev-proxy-with-github-actions
- **GitHub Repository**: https://github.com/dev-proxy-tools
- **MCP Package**: https://www.npmjs.com/package/@devproxy/mcp
- **Setup Action**: https://github.com/dev-proxy-tools/actions

### Security

- Dev Proxy runs locally in your workflow environment
- No external API calls are made unless configured to do so
- All proxy configurations are scoped to the workflow execution

