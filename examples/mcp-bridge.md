# MCP stdio-to-HTTP Bridge Examples

This file contains examples for using the `gh aw mcp bridge` command to convert stdio MCP servers to HTTP transport.

## Quick Start

The simplest way to test the bridge is with gh-aw's own MCP server:

```bash
# Terminal 1: Start the bridge
gh aw mcp bridge --command "gh" --args "aw,mcp-server" --port 8080

# Terminal 2: Test the HTTP endpoint
curl -i http://localhost:8080
```

## Example: Bridging a Custom MCP Server

If you have a custom MCP server that uses stdio transport:

```bash
# Bridge a Node.js MCP server
gh aw mcp bridge --command "npx" --args "@my/mcp-server" --port 3000

# Bridge a Python MCP server
gh aw mcp bridge --command "python" --args "my_server.py" --port 3000

# Bridge with multiple arguments
gh aw mcp bridge --command "node" --args "server.js,--verbose,--debug" --port 8080
```

## See Also

- [MCP Bridge Specification](../specs/mcp-bridge.md)
- [Model Context Protocol](https://modelcontextprotocol.io/)
