---
# Memory MCP Server (@modelcontextprotocol/server-memory)
# Docker-based MCP server for persistent memory storage
#
# Requires cache-memory to be enabled for persistent storage:
#   tools:
#     cache-memory: true
#
# Documentation: https://github.com/modelcontextprotocol/servers/tree/main/src/memory
#
# Available tools:
#   - store_memory: Store a piece of information with a key
#   - retrieve_memory: Retrieve stored information by key
#   - list_memories: List all stored memory keys
#   - delete_memory: Delete a memory by key
#
# Usage:
#   imports:
#     - shared/mcp/memory.md
#
# The memory server uses the cache-memory directory (/tmp/gh-aw/cache-memory/)
# for persistent storage across workflow runs.

mcp-servers:
  memory:
    container: "mcp/memory"
    args:
      - "-v"
      - "/tmp/gh-aw/cache-memory:/app/dist"
    allowed:
      - store_memory
      - retrieve_memory
      - list_memories
      - delete_memory
---

## Memory MCP Server

This shared configuration provides the @modelcontextprotocol/server-memory MCP server integration with persistent storage using the cache-memory directory.

### Available Tools

- `store_memory`: Store a piece of information with a key
- `retrieve_memory`: Retrieve stored information by key
- `list_memories`: List all stored memory keys
- `delete_memory`: Delete a memory by key

### Setup

1. Enable `cache-memory` in your workflow to provide persistent storage:

```yaml
tools:
  cache-memory: true
```

2. Import this configuration:

```yaml
imports:
  - shared/mcp/memory.md
```

### Example Usage

```aw
---
on: workflow_dispatch

tools:
  cache-memory: true

imports:
  - shared/mcp/memory.md

permissions:
  contents: read
---

# Memory Test Workflow

Test the memory MCP server by storing and retrieving information.

## Task

1. Store a test value using the memory server
2. Retrieve the stored value
3. List all stored memories
4. Report the results
```

### How It Works

The memory MCP server stores data in the `/tmp/gh-aw/cache-memory/` directory, which is made persistent across workflow runs through GitHub Actions cache. The Docker container mounts this directory to `/app/dist` inside the container where the memory server stores its data files.

### Persistence

Memory persistence is handled by the `cache-memory` configuration, which automatically:
- Creates the cache directory
- Restores previous cache data before workflow execution
- Saves cache data after workflow completion

This ensures that memories stored in one workflow run are available in subsequent runs.
