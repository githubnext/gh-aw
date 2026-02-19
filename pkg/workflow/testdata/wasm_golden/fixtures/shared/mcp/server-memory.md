---
# Server Memory MCP (@modelcontextprotocol/server-memory)
# Requires cache-memory: true for persistent storage

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

<!--
## Server Memory MCP

Provides @modelcontextprotocol/server-memory MCP server with persistent storage using cache-memory directory.

### Available Tools

- `store_memory`: Store information with a key
- `retrieve_memory`: Retrieve stored information
- `list_memories`: List all stored keys
- `delete_memory`: Delete a memory

### Setup

1. Enable cache-memory:
```yaml
tools:
  cache-memory: true
```

2. Import this configuration:
```yaml
imports:
  - shared/mcp/server-memory.md
```

### Example

```yaml
---
on: workflow_dispatch
tools:
  cache-memory: true
imports:
  - shared/mcp/server-memory.md
---

# Memory Workflow

Store and retrieve information across workflow runs using the memory server.
```

### How It Works

The memory MCP server stores data in `/tmp/gh-aw/cache-memory/`, which persists across runs via GitHub Actions cache. The Docker container mounts this directory to `/app/dist`.

Documentation: https://github.com/modelcontextprotocol/servers/tree/main/src/memory
-->
