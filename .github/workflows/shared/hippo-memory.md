---
# Hippo Memory - Shared Agentic Workflow Wrapper
# Provides persistent AI agent memory across runs using hippo-memory.
# The .hippo/ store is symlinked into cache-memory so learned lessons survive
# between workflow runs automatically.
#
# See: https://github.com/kitfunso/hippo-memory
#
# Usage:
#   runtimes:
#     node:
#       version: "22"         # hippo-memory requires Node.js 22.5+
#   network:
#     allowed:
#       - node                # Required for npm install -g hippo-memory
#   tools:
#     cache-memory: true      # REQUIRED: persists the .hippo store across runs
#   imports:
#     - shared/hippo-memory.md

tools:
  cache-memory: true

mcp-scripts:
  hippo:
    description: "Execute any hippo-memory CLI command. Accessible as 'mcpscripts-hippo'. Provide arguments after 'hippo'. Examples: args 'learn --git' to extract lessons from git commits, 'sleep' for full consolidation, 'recall \"api errors\" --budget 2000' to retrieve relevant memories."
    inputs:
      args:
        type: string
        description: "Arguments to pass to hippo CLI (without the 'hippo' prefix). Examples: 'learn --git', 'sleep', 'sleep --no-share', 'recall \"build failures\" --budget 3000', 'remember \"always run make fmt before committing\" --tag rule', 'list', 'export', 'export --format markdown'"
        required: true
    run: |
      echo "hippo $INPUT_ARGS"
      hippo $INPUT_ARGS

steps:
  - name: Install hippo-memory
    run: |
      npm install -g hippo-memory

  - name: Initialize hippo store
    run: |
      # Symlink .hippo into cache-memory so the SQLite store persists across runs.
      # All writes to .hippo/ land in /tmp/gh-aw/cache-memory/hippo-store/ and are
      # saved/restored automatically by the cache-memory mechanism.
      mkdir -p /tmp/gh-aw/cache-memory/hippo-store

      if [ ! -e ".hippo" ]; then
        ln -s /tmp/gh-aw/cache-memory/hippo-store .hippo
        echo "🔗 Created .hippo → cache-memory/hippo-store"
      elif [ -d ".hippo" ] && [ ! -L ".hippo" ]; then
        # Plain directory present (e.g. first run after adding this import) — migrate
        cp -r .hippo/. /tmp/gh-aw/cache-memory/hippo-store/ 2>/dev/null || true
        rm -rf .hippo
        ln -s /tmp/gh-aw/cache-memory/hippo-store .hippo
        echo "🔗 Migrated existing .hippo/ → cache-memory/hippo-store"
      else
        echo "✅ .hippo already linked to cache-memory/hippo-store"
      fi

      # Initialise if the store has never been set up, using --no-learn to avoid
      # a slow full-history git scan during setup.
      if [ ! -f ".hippo/config.json" ]; then
        hippo init --no-learn
        echo "✅ Hippo memory store initialised"
      else
        echo "✅ Hippo store restored from cache"
        hippo list 2>/dev/null | head -5 || true
      fi
---

**IMPORTANT**: Always use the `mcpscripts-hippo` tool for all hippo-memory commands.

## Hippo Memory Tools

Use the `mcpscripts-hippo` tool with the following command patterns:

### Learning from the Repository

```
mcpscripts-hippo args: "learn --git"          # Extract lessons from recent git commits
mcpscripts-hippo args: "sleep"                # Full cycle: learn, import MEMORY.md, dedup, share
mcpscripts-hippo args: "sleep --no-share"     # Consolidate without promoting to global store
```

### Recalling and Storing Memories

```
mcpscripts-hippo args: 'recall "build errors" --budget 3000'     # Retrieve relevant memories
mcpscripts-hippo args: 'remember "always run make fmt" --tag rule'  # Store a new memory
mcpscripts-hippo args: 'list'                                     # List all memories
mcpscripts-hippo args: 'export'                                   # Export all memories as JSON
mcpscripts-hippo args: 'export --format markdown'                 # Export as markdown
```

### Inspection and Session State

```
mcpscripts-hippo args: 'current show'          # Show active session context
mcpscripts-hippo args: 'inspect <id>'          # Inspect a specific memory entry
mcpscripts-hippo args: 'last-sleep'            # Show output of the previous sleep run
```

## Persistence

The `.hippo/` store is symlinked to `/tmp/gh-aw/cache-memory/hippo-store/` so the
SQLite index and YAML mirrors are automatically saved and restored across workflow
runs via `cache-memory`.

## Requirements

The importing workflow must provide:
- `runtimes.node.version: "22"` — hippo-memory requires Node.js 22.5+
- `node` in `network.allowed` — needed to `npm install -g hippo-memory`
- `tools.cache-memory: true` — already set by this import, but ensure it is not disabled
