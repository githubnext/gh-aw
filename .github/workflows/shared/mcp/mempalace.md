---
# MemPalace MCP Server
# Semantic memory system for AI agents — store, search, and retrieve
# verbatim content across workflow runs using ChromaDB.
#
# Palace data is persisted using cache-memory across workflow runs.
# On first use the palace directory is created automatically.
#
# Documentation: https://github.com/MemPalace/mempalace
#
# Usage:
#   imports:
#     - shared/mcp/mempalace.md

tools:
  cache-memory: true

mcp-servers:
  mempalace:
    type: http
    url: http://localhost:8765/mcp
    allowed:
      - "mempalace_status"
      - "mempalace_list_wings"
      - "mempalace_list_rooms"
      - "mempalace_get_taxonomy"
      - "mempalace_search"
      - "mempalace_check_duplicate"
      - "mempalace_get_aaak_spec"
      - "mempalace_add_drawer"
      - "mempalace_delete_drawer"
      - "mempalace_kg_query"
      - "mempalace_kg_add"
      - "mempalace_kg_invalidate"
      - "mempalace_kg_timeline"
      - "mempalace_kg_stats"
      - "mempalace_traverse"
      - "mempalace_find_tunnels"
      - "mempalace_graph_stats"
      - "mempalace_diary_write"
      - "mempalace_diary_read"

steps:
  - name: Install MemPalace
    run: pip install "mempalace==3.2.0"
  - name: Start MemPalace MCP server
    run: |
      set -e
      mkdir -p /tmp/gh-aw/mcp-logs/mempalace
      mkdir -p /tmp/gh-aw/cache-memory/palace
      python -m mempalace.mcp_server \
        --palace /tmp/gh-aw/cache-memory/palace \
        --transport streamable-http \
        --host 127.0.0.1 \
        --port 8765 \
        > /tmp/gh-aw/mcp-logs/mempalace/server.log 2>&1 &
      echo "MemPalace MCP server started (PID $!)"
---
<!--
## MemPalace Memory System

Provides the MemPalace MCP server with 19 tools for storing and retrieving
verbatim content in a ChromaDB palace backed by GitHub Actions cache-memory.

### Setup

Import this configuration:

```yaml
imports:
  - shared/mcp/mempalace.md
```

The palace data is persisted at `/tmp/gh-aw/cache-memory/palace/` across runs
using GitHub Actions cache. By default cache expires after 7 days; increase
`retention-days` to extend up to 90 days:

```yaml
tools:
  cache-memory:
    retention-days: 90
```

### Available Tools

**Palace (read)**
- `mempalace_status` — Palace overview, AAAK spec, and memory protocol
- `mempalace_list_wings` — Wings with drawer counts
- `mempalace_list_rooms` — Rooms within a wing
- `mempalace_get_taxonomy` — Full wing → room → count tree
- `mempalace_search` — Semantic search with optional wing/room filters
- `mempalace_check_duplicate` — Check for existing content before filing
- `mempalace_get_aaak_spec` — AAAK dialect reference

**Palace (write)**
- `mempalace_add_drawer` — File verbatim content into a wing/room
- `mempalace_delete_drawer` — Remove a drawer by ID

**Knowledge Graph**
- `mempalace_kg_query` — Entity relationships with time filtering
- `mempalace_kg_add` — Add facts to the knowledge graph
- `mempalace_kg_invalidate` — Mark facts as ended/superseded
- `mempalace_kg_timeline` — Chronological entity story
- `mempalace_kg_stats` — Knowledge graph overview

**Navigation**
- `mempalace_traverse` — Walk the graph from a room across wings
- `mempalace_find_tunnels` — Find rooms bridging two wings
- `mempalace_graph_stats` — Graph connectivity overview

**Agent Diary**
- `mempalace_diary_write` — Write an AAAK diary entry
- `mempalace_diary_read` — Read recent diary entries

### First Run

On first use the palace is created automatically by ChromaDB at
`/tmp/gh-aw/cache-memory/palace/`. Call `mempalace_status` to verify
the palace is initialized and check the memory protocol.

### Example Workflow

```yaml
---
engine: copilot
imports:
  - shared/mcp/mempalace.md
tools:
  cache-memory:
    retention-days: 30
---

Search the palace for previous decisions about authentication:
use mempalace_search with query "authentication decisions".

If relevant memories are found, use them as context for the current task.
After completing work, store a new memory using mempalace_add_drawer.
```
-->
