# MCP Gateway Command

The `gh aw mcp-gateway` command implements a minimal MCP proxy application that aggregates multiple MCP servers into a single HTTP gateway.

## Features

- **Integrates with sandbox.mcp**: Works with the `sandbox.mcp` extension point in workflows
- **Multiple MCP servers**: Supports connecting to multiple MCP servers simultaneously
- **MCP protocol support**: Implements `initialize`, `list_tools`, `call_tool`, `list_resources`, `list_prompts`
- **Transport support**: Currently supports stdio/command transport, HTTP transport planned
- **Comprehensive logging**: Logs to file in MCP log directory (`/tmp/gh-aw/mcp-gateway-logs` by default)
- **API key authentication**: Optional API key for securing gateway endpoints

## Usage

### Basic Usage

```bash
# From stdin (reads JSON config from standard input)
echo '{"mcpServers":{"gh-aw":{"command":"gh","args":["aw","mcp-server"]}}}' | gh aw mcp-gateway

# From config file
gh aw mcp-gateway --config config.json

# Custom port and log directory
gh aw mcp-gateway --config config.json --port 8088 --log-dir /custom/logs
```

### Configuration Format

The gateway accepts configuration in JSON format:

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
    "http-server": {
      "url": "http://localhost:3000"
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "optional-api-key"
  }
}
