# awmg - Agentic Workflows MCP Gateway

`awmg` is a standalone binary that implements an MCP (Model Context Protocol) gateway for aggregating multiple MCP servers into a single HTTP endpoint.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/githubnext/gh-aw.git
cd gh-aw

# Build the binary
make build-awmg

# The binary will be created as ./awmg
```

### Pre-built Binaries

Download the latest release from the [GitHub releases page](https://github.com/githubnext/gh-aw/releases).

## Usage

```bash
# Start gateway with config file
awmg --config config.json

# Start gateway reading from stdin
echo '{"mcpServers":{...}}' | awmg --port 8080

# Custom log directory
awmg --config config.json --log-dir /var/log/mcp-gateway
```

## Configuration

The gateway accepts JSON configuration with the following format:

```json
{
  "mcpServers": {
    "server-name": {
      "command": "command-to-run",
      "args": ["arg1", "arg2"],
      "env": {
        "ENV_VAR": "value"
      }
    },
    "another-server": {
      "url": "http://localhost:3000"
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "optional-api-key"
  }
}
```

### Configuration Fields

- `mcpServers`: Map of MCP server configurations
  - Each server can be configured with:
    - `command`: Command to execute (for stdio transport)
    - `args`: Command arguments
    - `env`: Environment variables
    - `url`: HTTP URL (for HTTP transport)
- `gateway`: Gateway-specific settings
  - `port`: HTTP port (default: 8080)
  - `apiKey`: Optional API key for authentication

## Endpoints

Once running, the gateway exposes the following HTTP endpoints:

- `GET /health` - Health check endpoint
- `GET /servers` - List all configured MCP servers
- `POST /mcp/{server}` - Proxy MCP requests to a specific server

## Examples

### Example 1: Single gh-aw MCP Server

```json
{
  "mcpServers": {
    "gh-aw": {
      "command": "gh",
      "args": ["aw", "mcp-server"]
    }
  },
  "gateway": {
    "port": 8088
  }
}
```

### Example 2: Multiple Servers

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
    }
  },
  "gateway": {
    "port": 8088
  }
}
```

## Integration with GitHub Agentic Workflows

The awmg binary is designed to work seamlessly with GitHub Agentic Workflows. When you configure `sandbox.mcp` in your workflow, the system automatically sets up the MCP gateway:

```yaml
---
sandbox:
  mcp:
    # MCP gateway runs as standalone awmg CLI
    port: 8080
---
```

## Features

- ✅ **Multiple MCP Servers**: Connect to and manage multiple MCP servers
- ✅ **HTTP Gateway**: Expose all servers through a unified HTTP interface
- ✅ **Protocol Support**: Supports initialize, list_tools, call_tool, list_resources, list_prompts
- ✅ **Comprehensive Logging**: Per-server log files with detailed operation logs
- ✅ **Command Transport**: Subprocess-based MCP servers via stdio
- ⏳ **HTTP Transport**: HTTP/SSE transport (planned)
- ⏳ **Docker Support**: Container-based MCP servers (planned)

## Development

```bash
# Run tests
make test

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

## See Also

- [MCP Gateway Specification](../specs/mcp-gateway.md)
- [MCP Gateway Usage Guide](mcp-gateway.md)
- [GitHub Agentic Workflows Documentation](https://github.com/githubnext/gh-aw)
