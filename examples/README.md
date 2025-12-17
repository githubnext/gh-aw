# Examples

This directory contains example configurations and templates for GitHub Agentic Workflows.

## Secure Workflow Template

**[`secure-workflow-template.md`](secure-workflow-template.md)** - A comprehensive template demonstrating security best practices for creating new workflows. Use this as a starting point for any new workflow to ensure security is built-in from the start.

Key security features:
- Explicit permissions with principle of least privilege
- Strict mode enforcement
- Safe outputs for write operations
- Threat detection
- Network isolation
- Tool hardening with explicit allow-lists
- Sanitized context usage

## MCP Gateway Examples

This directory also contains example configuration files for the `mcp-gateway` command.

## What is MCP Gateway?

The MCP Gateway is a proxy server that connects to multiple MCP (Model Context Protocol) servers and exposes all their tools through a single HTTP endpoint. This allows clients to access tools from multiple MCP servers without managing individual connections.

## Example Configurations

### Simple Configuration (`mcp-gateway-config.json`)

A basic configuration with a single MCP server:

```json
{
  "mcpServers": {
    "gh-aw": {
      "command": "gh",
      "args": ["aw", "mcp-server"]
    }
  },
  "port": 8088
}
```

**Note:** The `port` field is optional in the configuration file. If not specified, the gateway will use port 8088 by default, or you can override it with the `--port` flag.

### Multi-Server Configuration (`mcp-gateway-multi-server.json`)

A more complex configuration demonstrating all three server types:

```json
{
  "mcpServers": {
    "gh-aw": {
      "command": "gh",
      "args": ["aw", "mcp-server"],
      "env": {
        "DEBUG": "cli:*"
      }
    },
    "remote-server": {
      "url": "http://localhost:3000"
    },
    "docker-server": {
      "container": "mcp-server:latest",
      "args": ["--verbose"],
      "env": {
        "LOG_LEVEL": "debug"
      }
    }
  },
  "port": 8088
}
```

## Server Types

### Stdio Servers

Use the `command` field to specify a command-line MCP server:

```json
{
  "command": "node",
  "args": ["server.js"],
  "env": {
    "ENV_VAR": "value"
  }
}
```

### HTTP Servers

Use the `url` field to connect to an HTTP MCP server:

```json
{
  "url": "http://localhost:3000"
}
```

### Docker Servers

Use the `container` field to run an MCP server in a Docker container:

```json
{
  "container": "my-mcp-server:latest",
  "args": ["--option", "value"],
  "env": {
    "ENV_VAR": "value"
  }
}
```

## Usage

### Start the Gateway

```bash
# Use default port 8088
gh aw mcp-gateway mcp-gateway-config.json

# Specify a custom port
gh aw mcp-gateway --port 9000 mcp-gateway-config.json
```

### Enable API Key Authentication

```bash
gh aw mcp-gateway --api-key secret123 mcp-gateway-config.json
```

When API key authentication is enabled, clients must include the API key in the `Authorization` header:

```bash
curl -H "Authorization: ******" http://localhost:8088/...
# or
curl -H "Authorization: secret123" http://localhost:8088/...
```

### Write Debug Logs to File

```bash
gh aw mcp-gateway --logs-dir /tmp/gateway-logs mcp-gateway-config.json
```

This creates the specified directory and prepares it for logging output.

### Combined Example

```bash
gh aw mcp-gateway \
  --port 9000 \
  --api-key mySecretKey \
  --logs-dir /var/log/mcp-gateway \
  mcp-gateway-config.json
```

### Enable Verbose Logging

```bash
DEBUG=* gh aw mcp-gateway mcp-gateway-config.json
```

Or for specific modules:

```bash
DEBUG=pkg:gateway gh aw mcp-gateway mcp-gateway-config.json
```

## How It Works

1. **Startup**: The gateway connects to all configured MCP servers
2. **Tool Discovery**: It lists all available tools from each server
3. **Name Resolution**: If tool names conflict, they're prefixed with the server name (e.g., `server1.tool-name`)
4. **HTTP Server**: An HTTP MCP server starts on the configured port
5. **Proxying**: Tool calls are routed to the appropriate backend server
6. **Response**: Results are returned to the client

## Use Cases

- **Unified Interface**: Access tools from multiple MCP servers through a single endpoint
- **Development**: Test multiple MCP servers together
- **Sandboxing**: Act as a gateway for MCP servers with the `sandbox.mcp` configuration
- **Tool Aggregation**: Combine tools from different sources into one interface

## Troubleshooting

### Connection Errors

If a server fails to connect, the gateway will log the error and continue with other servers:

```
✗ failed to connect to MCP servers: failed to connect to some servers: [server test: failed to connect: calling "initialize": EOF]
```

### Port Already in Use

If the port is already in use, try a different port:

```bash
gh aw mcp-gateway --port 8081 mcp-gateway-config.json
```

### Tool Name Collisions

If multiple servers expose tools with the same name, the gateway automatically prefixes them:

- Original: `status` from `server1` and `server2`
- Result: `status` (first server) and `server2.status` (second server)
