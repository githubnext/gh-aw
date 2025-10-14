# Drain3 MCP Server

This directory contains the Docker-based MCP server implementation for Drain3, a log template miner.

## Files

- **`drain3_mcp_server.py`**: Python MCP server script that wraps Drain3
- **`Dockerfile`**: Docker image definition for the MCP server
- **`README.md`**: This file

## Building

To build the Docker image:

```bash
docker build -t mcp/drain3 .
```

## Running

The Docker image is designed to run as an MCP server using stdio transport:

```bash
docker run -i mcp/drain3
```

## Testing

To test the server, you can use an MCP client or test the tools directly:

```bash
# Example: Parse a log line
# (This requires an MCP client to properly communicate with the server)
docker run -i mcp/drain3
```

## Usage in Workflows

To use this MCP server in an agentic workflow, include the shared configuration:

```yaml
imports:
  - shared/mcp/drain3.md
```

See the parent `drain3.md` file for complete usage examples.
