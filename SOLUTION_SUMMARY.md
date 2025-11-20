# MCP stdio-to-HTTP Bridge Solution

## Problem Statement

MCP stdio servers require the agent process to spawn them as subprocesses, which is not ideal for isolation purposes. The goal was to find a lightweight way to convert MCP servers with stdio to HTTP transport.

## Solution Overview

Implemented a new `gh aw mcp bridge` command that provides a lightweight stdio-to-HTTP bridge for MCP servers.

## Implementation Details

### Architecture

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

1. **Initialization**: Starts the stdio MCP server once as a subprocess using `CommandTransport`
2. **Discovery**: Queries the stdio server for capabilities:
   - `ListTools()` - Discovers all available tools
   - `ListPrompts()` - Discovers all available prompts
   - `ListResources()` - Discovers all available resources
3. **Proxy Setup**: Creates an MCP server that mirrors these capabilities
4. **HTTP Server**: Exposes proxy via `StreamableHTTPHandler` with SSE transport
5. **Request Forwarding**: All client requests are forwarded to the stdio server

### Key Components

1. **`pkg/cli/mcp_bridge.go`** (215 lines)
   - Command-line interface
   - stdio server lifecycle management
   - Dynamic capability discovery
   - HTTP proxy server creation
   - Request forwarding logic

2. **`pkg/cli/mcp_bridge_test.go`**
   - Unit tests for command structure
   - Flag validation tests
   - Help text verification

3. **`pkg/cli/mcp_bridge_integration_test.go`**
   - End-to-end integration test
   - Tests bridging gh-aw's own MCP server
   - Verifies HTTP endpoint accessibility

4. **`specs/mcp-bridge.md`**
   - Technical specification
   - Architecture documentation
   - Benefits and use cases

5. **`examples/mcp-bridge.md`**
   - Usage examples
   - Quick start guide
   - Troubleshooting tips

## Usage

### Basic Example

```bash
# Bridge gh-aw's own MCP server
gh aw mcp bridge --command "gh" --args "aw,mcp-server" --port 8080
```

### With Custom Server

```bash
# Bridge a Node.js MCP server
gh aw mcp bridge --command "npx" --args "@my/mcp-server" --port 3000
```

### In Workflows

```yaml
tools:
  my-custom-mcp:
    type: http
    url: "http://localhost:8080"
```

## Benefits

### Process Isolation
- Single stdio server process shared across all HTTP clients
- Reduces resource usage (one subprocess vs many)
- Better security through process separation

### Network Accessibility
- stdio servers become accessible over HTTP
- Enables remote access and API integration
- Compatible with load balancers and reverse proxies

### Zero Modification
- Works with any existing stdio MCP server
- No changes needed to server code
- Transparent proxy - clients see same capabilities

### Resource Efficiency
- One subprocess instead of one per client
- Shared connection to stdio server
- Lower memory and CPU overhead

## Technical Details

### MCP SDK Usage

- **Go SDK**: `github.com/modelcontextprotocol/go-sdk v1.1.0`
- **CommandTransport**: For stdio subprocess communication
- **StreamableHTTPHandler**: For HTTP/SSE endpoint
- **Dynamic Registration**: Tools/prompts/resources discovered at runtime

### Discovery Process

```go
// List all tools from stdio server
toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})

// Register each tool with forwarding handler
for _, tool := range toolsResult.Tools {
    server.AddTool(tool, func(ctx context.Context, req *mcp.ServerRequest[*mcp.CallToolParamsRaw]) (*mcp.CallToolResult, error) {
        return session.CallTool(ctx, &mcp.CallToolParams{
            Name:      tool.Name,
            Arguments: req.Params.Arguments,
        })
    })
}
```

### Request Flow

1. HTTP client connects to bridge endpoint
2. Client sends MCP request via SSE
3. Bridge forwards request to stdio server
4. stdio server processes request
5. Bridge returns response to HTTP client

## Testing

### Unit Tests
- Command structure validation
- Flag parsing
- Help text verification

### Integration Tests
- Bridges gh-aw's MCP server
- Verifies tool discovery (7 tools found)
- Tests HTTP endpoint accessibility
- Validates subprocess management

### Test Results
```
✅ All unit tests pass
✅ Integration test successfully bridges gh-aw mcp-server
✅ Discovers 7 tools from stdio server
✅ HTTP endpoint accessible
✅ Code formatting validated
✅ Linting passes
```

## Comparison with Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **stdio-to-HTTP Bridge** (This solution) | No server modification, single subprocess, dynamic discovery | Requires bridge process |
| **Native HTTP Server** | No bridge needed, optimal performance | Requires server rewrite |
| **Per-Client stdio** | Simple implementation | Poor isolation, high resource usage |
| **Containerized stdio** | Good isolation | Complex orchestration, slow startup |

## Future Enhancements

Potential improvements:
- Authentication/authorization middleware
- Multiple backend stdio servers (load balancing)
- Connection pooling
- Metrics and monitoring
- TLS/HTTPS support
- Configuration file support

## References

- [MCP Specification](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
- [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)

## Conclusion

The `gh aw mcp bridge` command provides a practical, lightweight solution for converting stdio MCP servers to HTTP transport. It addresses the process isolation problem while maintaining compatibility with existing stdio servers and requiring zero modifications to server code.
