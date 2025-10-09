---
mcp-servers:
  everything:
    container: "mcp/everything"
    version: "latest"
    allowed: ["*"]
---

## Everything MCP Server

The Everything MCP server is an aggregated MCP server that combines multiple common MCP tools into a single container for convenience. It provides a comprehensive set of tools for various tasks without needing to configure individual MCP servers.

### Available Tools

The Everything MCP server includes tools from multiple MCP servers. The specific tools available can be discovered using the MCP protocol. This server typically includes:
- File system operations
- Web content fetching
- Database interactions
- And other common MCP functionality

### Usage

Import this shared component in your workflow:

```yaml
imports:
  - shared/everything-mcp.md
```

### More Information

- Docker image: https://hub.docker.com/r/mcp/everything
- MCP documentation: https://modelcontextprotocol.io/

### Configuration

This configuration uses:
- **Container**: `mcp/everything:latest` from Docker Hub
- **Allowed tools**: All tools (`*`) - for maximum convenience
- **No authentication required** - public container

For production workflows, consider:
1. Pinning to a specific version instead of `latest`
2. Limiting `allowed` to only the tools you need for better security
3. Adding network restrictions if the tools access external services
