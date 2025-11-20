# MCP stdio-to-HTTP Bridge

## Overview

The `gh aw mcp bridge` command provides a lightweight solution for converting stdio-based MCP (Model Context Protocol) servers to HTTP transport with SSE (Server-Sent Events). This enables better process isolation and network accessibility for MCP servers.

## Problem

stdio-based MCP servers require spawning as subprocesses, which presents several challenges:
- **Process isolation**: Each client must spawn its own subprocess
- **Resource management**: Multiple clients = multiple processes
- **Network access**: stdio servers can't be accessed over the network
- **Containerization**: Difficult to run in restricted environments

## Solution

The MCP bridge acts as a proxy that:
1. Starts a single stdio MCP server as a subprocess
2. Exposes an HTTP endpoint with SSE transport
3. Forwards all requests to the stdio server
4. Manages the lifecycle of the stdio server process

## Usage

### Basic Example

```bash
# Bridge a Node.js MCP server
gh aw mcp bridge --command "npx" --args "@my/mcp-server" --port 8080
```

### With Multiple Arguments

```bash
# Bridge a Python server with arguments
gh aw mcp bridge --command "python" --args "server.py,--verbose,--debug" --port 3000
```

### Custom Server

```bash
# Bridge a custom executable
gh aw mcp bridge --command "./my-server" --port 9000
```

## Architecture

```
┌─────────────┐      HTTP/SSE       ┌──────────────┐      stdio      ┌──────────────┐
│ HTTP Client │ ◄─────────────────► │ MCP Bridge   │ ◄──────────────► │ stdio Server │
└─────────────┘                     └──────────────┘                  └──────────────┘
                                           │
                                           │ Discovery & Proxy
                                           │
                                    ┌──────▼───────┐
                                    │ Proxy Server │
                                    │ - Tools      │
                                    │ - Prompts    │
                                    │ - Resources  │
                                    └──────────────┘
```

### How It Works

1. **Initialization**: The bridge starts the stdio server and connects to it using the MCP Go SDK
2. **Discovery**: It queries the stdio server for all available tools, prompts, and resources
3. **Proxy Setup**: Creates a proxy MCP server that mirrors the stdio server's capabilities
4. **HTTP Server**: Exposes the proxy server via HTTP with SSE transport
5. **Request Forwarding**: All client requests are forwarded to the stdio server

## Benefits

### Process Isolation
- Single stdio server process shared across all HTTP clients
- Reduces resource usage compared to per-client subprocesses
- Better security through process separation

### Network Accessibility
- stdio servers become accessible over HTTP
- Enables remote access and API integration
- Compatible with load balancers and reverse proxies

### Containerization
- Easier to deploy in Docker/Kubernetes
- No subprocess spawning required by clients
- Better resource limits and monitoring

### Compatibility
- Works with any stdio MCP server
- No modifications needed to existing servers
- Transparent proxy - clients see the same capabilities

## Command Reference

### Flags

- `--command, -c` (required): Command to run the stdio MCP server
- `--port, -p` (required): Port to run HTTP server on
- `--args, -a` (optional): Comma-separated arguments for the server command

### Examples

#### Bridging the gh-aw MCP Server

```bash
# Run gh-aw's own MCP server via bridge
gh aw mcp bridge --command "gh" --args "aw,mcp-server" --port 8080
```

Then connect via HTTP:
```bash
curl http://localhost:8080
```

#### Bridging a Custom Server

```bash
# Bridge a custom Python MCP server
gh aw mcp bridge --command "python3" --args "my_mcp_server.py" --port 3000
```

## Limitations

- Single stdio server instance (no horizontal scaling)
- Connection state shared across all HTTP clients
- Requires stdio server to be stateless or handle concurrent requests
- No built-in authentication (use reverse proxy for auth)

## Integration with Agentic Workflows

While the bridge is primarily a development/deployment tool, it can be referenced in workflow configurations:

```yaml
tools:
  my-custom-mcp:
    type: http
    url: "http://localhost:8080"
```

## Comparison with Direct HTTP MCP Servers

| Feature | stdio Bridge | Native HTTP MCP |
|---------|-------------|-----------------|
| Deployment | Requires bridge command | Direct HTTP server |
| Process Model | Single subprocess + bridge | Single HTTP server |
| Compatibility | Any stdio server | HTTP-only servers |
| Overhead | Minimal proxying | None |
| Use Case | Convert existing stdio | Purpose-built HTTP |

## Implementation Details

The bridge uses the MCP Go SDK (v1.1.0) which provides:
- `CommandTransport` for stdio communication
- `StreamableHTTPHandler` for HTTP/SSE endpoints
- Protocol-level request/response handling
- Tool, prompt, and resource registration

Key implementation points:
- Dynamic tool discovery via `ListTools`, `ListPrompts`, `ListResources`
- Request forwarding using `CallTool`, `GetPrompt`, `ReadResource`
- Single stdio server process shared across HTTP connections
- Automatic capability mirroring from stdio to HTTP

## Future Enhancements

Potential improvements:
- Authentication and authorization support
- Multiple backend stdio servers (load balancing)
- Connection pooling and request queuing
- Metrics and monitoring endpoints
- TLS/HTTPS support
- Configuration file support

## Related Commands

- `gh aw mcp-server`: Run gh-aw as an MCP server (stdio or HTTP)
- `gh aw mcp inspect`: Inspect MCP servers in workflows
- `gh aw mcp add`: Add MCP servers to workflows

## References

- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [MCP Go SDK Documentation](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
- [Server-Sent Events (SSE)](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
