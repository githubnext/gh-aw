# MCP Gateway Examples

This directory contains example configurations for the MCP Gateway with safe-inputs support.

## Safe-Inputs Examples

The `mcp-gateway-tools.json` file demonstrates how to configure custom tools using JavaScript, Python, and Shell handlers.

### Running the Examples

1. Start the gateway with safe-inputs:

```bash
gh aw mcp-gateway --scripts examples/mcp-gateway-tools.json --port 8090
```

2. The gateway will expose three tools:
   - `greet` (JavaScript) - Greets a person by name
   - `calculate` (Python) - Calculates the sum of two numbers
   - `system_info` (Shell) - Gets system information

### Tool Handlers

- **greet.cjs**: JavaScript handler that demonstrates async/await and error handling
- **calculate.py**: Python handler that reads JSON input from stdin and outputs JSON
- **system_info.sh**: Shell handler that uses GitHub Actions conventions (GITHUB_OUTPUT)

### MCP Servers Configuration

The existing `mcp-gateway-config.json` demonstrates how to proxy to external MCP servers:

```bash
gh aw mcp-gateway --mcps examples/mcp-gateway-config.json
```

### Combined Mode

You can run both MCP servers and safe-inputs together:

```bash
gh aw mcp-gateway \
  --mcps examples/mcp-gateway-config.json \
  --scripts examples/mcp-gateway-tools.json \
  --port 8090
```

## Tool Configuration Format

The `mcp-gateway-tools.json` follows this structure:

```json
{
  "serverName": "example-safeinputs",
  "version": "1.0.0",
  "tools": [
    {
      "name": "tool_name",
      "description": "Tool description",
      "inputSchema": {
        "type": "object",
        "properties": {
          "param": {
            "type": "string",
            "description": "Parameter description"
          }
        },
        "required": ["param"]
      },
      "handler": "tool_handler.cjs"
    }
  ]
}
```

### Handler Types

- **.cjs**: JavaScript/Node.js handlers
  - Must export an `async function execute(inputs)` 
  - Returns any JSON-serializable value
  
- **.py**: Python handlers
  - Reads JSON input from stdin
  - Outputs JSON result to stdout
  
- **.sh**: Shell script handlers
  - Reads inputs from `INPUT_*` environment variables
  - Writes outputs to `$GITHUB_OUTPUT` file

## API Key Authentication

Add API key authentication to secure your gateway:

```bash
gh aw mcp-gateway \
  --scripts examples/mcp-gateway-tools.json \
  --api-key your-secret-key \
  --port 8090
```

Clients must include the API key in the Authorization header:
```
Authorization: Bearer your-secret-key
```
