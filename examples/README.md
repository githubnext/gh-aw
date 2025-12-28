# Examples

This directory contains example workflows and configurations for GitHub Agentic Workflows.

## Network Configuration Examples

For examples of network configuration with package registries and CDNs:
- [`network-python-project.md`](./network-python-project.md) - Python project with PyPI access
- [`network-node-project.md`](./network-node-project.md) - Node.js project with npm access
- [`network-multi-language.md`](./network-multi-language.md) - Multi-language project with multiple registries

See the [Network Configuration Guide](../docs/src/content/docs/guides/network-configuration.md) for more information.

## Model Context Protocol (MCP) Gateway Examples

This directory also contains MCP Gateway configuration files for the `mcp-gateway` command.

## What is MCP Gateway?

The MCP Gateway is a proxy server that connects to multiple Model Context Protocol (MCP) servers and exposes all their tools through a single HTTP endpoint. This allows clients to access tools from multiple MCP servers without managing individual connections.

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
```text

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
```text

### Multi-Config Example

Use multiple configuration files that are merged together:

**Base Configuration (`mcp-gateway-base.json`)** - Common servers:
```json
{
  "mcpServers": {
    "gh-aw": {
      "command": "gh",
      "args": ["aw", "mcp-server"]
    },
    "time": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-time"]
    }
  },
  "gateway": {
    "port": 8088
  }
}
```text

**Override Configuration (`mcp-gateway-override.json`)** - Environment-specific overrides:
```json
{
  "mcpServers": {
    "time": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-time"],
      "env": {
        "DEBUG": "mcp:*"
      }
    },
    "memory": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-memory"]
    }
  },
  "gateway": {
    "port": 9090,
    "apiKey": "optional-api-key"
  }
}
```text

**Usage:**
```bash
awmg --config mcp-gateway-base.json --config mcp-gateway-override.json
```text

**Result:** The merged configuration will have:
- `gh-aw` server (from base)
- `time` server with debug environment variable (overridden from override)
- `memory` server (added from override)
- Port 9090 and API key (overridden from override)

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
```text

### HTTP Servers

Use the `url` field to connect to an HTTP MCP server:

```json
{
  "url": "http://localhost:3000"
}
```text

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
```text

## Usage

### Start the Gateway

```bash
# From a single config file
awmg --config mcp-gateway-config.json

# From multiple config files (merged in order)
awmg --config base-config.json --config override-config.json

# Specify a custom port
awmg --config mcp-gateway-config.json --port 9000
```text

### Multiple Configuration Files

The gateway supports loading multiple configuration files which are merged in order. Later files override settings from earlier files:

```bash
# Base configuration with common servers
awmg --config common-servers.json --config team-specific.json

# Add environment-specific overrides
awmg --config base.json --config staging.json
```text

**Merge Behavior:**
- **MCP Servers**: Later configurations override servers with the same name
- **Gateway Settings**: Later configurations override gateway port and API key (if specified)
- **Example**: If `base.json` defines `server1` and `server2`, and `override.json` redefines `server2` and adds `server3`, the result will have all three servers with `server2` coming from `override.json`

### Enable API Key Authentication

```bash
awmg --config mcp-gateway-config.json --api-key secret123
```text

When API key authentication is enabled, clients must include the API key in the `Authorization` header:

```bash
curl -H "Authorization: Bearer secret123" http://localhost:8088/...
```text

### Write Debug Logs to File

```bash
awmg --config mcp-gateway-config.json --log-dir /tmp/gateway-logs
```text

This creates the specified directory and prepares it for logging output.

### Combined Example

```bash
awmg \
  --config base-config.json \
  --config override-config.json \
  --port 9000 \
  --api-key mySecretKey \
  --log-dir /var/log/mcp-gateway
```text

### Enable Verbose Logging

```bash
DEBUG=* awmg --config mcp-gateway-config.json
```text

Or for specific modules:

```bash
DEBUG=cli:mcp_gateway awmg --config mcp-gateway-config.json
```text

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

```text
✗ failed to connect to MCP servers: failed to connect to some servers: [server test: failed to connect: calling "initialize": EOF]
```text

### Port Already in Use

If the port is already in use, try a different port:

```bash
gh aw mcp-gateway --port 8081 mcp-gateway-config.json
```text

### Tool Name Collisions

If multiple servers expose tools with the same name, the gateway automatically prefixes them:

- Original: `status` from `server1` and `server2`
- Result: `status` (first server) and `server2.status` (second server)
