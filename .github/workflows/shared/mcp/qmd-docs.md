---
# QMD MCP Server
# Local on-device search engine for the project documentation, agents, and workflow instructions
#
# Documentation: https://github.com/tobi/qmd
#
# Available tools (via MCP):
#   - query: Search with typed sub-queries (lex/vec/hyde), combined via RRF + reranking
#   - get: Retrieve a document by path or docid (with fuzzy matching suggestions)
#   - multi_get: Batch retrieve by glob pattern, comma-separated list, or docids
#   - status: Index health and collection info
#
# Usage:
#   imports:
#     - shared/mcp/qmd-docs.md

mcp-servers:
  qmd:
    type: http
    url: http://localhost:8181/mcp
    allowed:
      - query
      - get
      - multi_get
      - status

resources:
  - .github/workflows/qmd-docs-indexer.yml

steps:
  - name: Setup Node.js
    uses: actions/setup-node@v6.3.0
    with:
      node-version: "24"
  - name: Install QMD
    run: npm install -g @tobilu/qmd
  - name: Restore QMD index cache
    uses: actions/cache/restore@v5.0.3
    with:
      path: ~/.cache/qmd
      key: qmd-docs-${{ hashFiles('docs/src/content/docs/**', '.github/agents/**', '.github/aw/**') }}
      restore-keys: qmd-docs-
  - name: Start QMD MCP server
    run: |
      set -e
      mkdir -p /tmp/gh-aw/mcp-logs/qmd/

      # Re-register collections so qmd MCP tools are available.
      # The cache restore only restores the index data; qmd requires collections
      # to be registered in the current session for MCP tools to be served.
      DOCS_DIR="${GITHUB_WORKSPACE}/docs/src/content/docs"
      AGENTS_DIR="${GITHUB_WORKSPACE}/.github/agents"
      AW_DIR="${GITHUB_WORKSPACE}/.github/aw"

      [ -d "$DOCS_DIR" ]   && qmd collection add "$DOCS_DIR"   --name docs   2>/dev/null || true
      [ -d "$AGENTS_DIR" ] && qmd collection add "$AGENTS_DIR" --name agents 2>/dev/null || true
      [ -d "$AW_DIR" ]     && qmd collection add "$AW_DIR"     --name aw     2>/dev/null || true

      # Start QMD MCP server in the background (explicit nohup rather than
      # --daemon to ensure the process survives the step and stays running)
      nohup qmd mcp --http > /tmp/gh-aw/mcp-logs/qmd/server.log 2>&1 &
      echo $! > /tmp/gh-aw/mcp-logs/qmd/server.pid
      echo "QMD MCP server started (PID $(cat /tmp/gh-aw/mcp-logs/qmd/server.pid))"

      # Poll until the server is healthy (up to 15 seconds)
      for i in $(seq 1 30); do
        if curl -sf http://localhost:8181/health > /dev/null 2>&1; then
          echo "QMD MCP server is healthy"
          echo "Status: $(curl -s http://localhost:8181/health)"
          break
        fi
        sleep 0.5
      done

      if ! curl -sf http://localhost:8181/health > /dev/null 2>&1; then
        echo "QMD MCP server health check timed out after 15 seconds"
        echo "Server logs:"
        cat /tmp/gh-aw/mcp-logs/qmd/server.log || true
        exit 1
      fi

      # Verify the MCP tools endpoint is responding with actual tools.
      # Without this check, a server with no registered collections will pass
      # the health check but return an empty tools list to the MCP Gateway.
      for i in $(seq 1 20); do
        TOOLS=$(curl -sf -X POST http://localhost:8181/mcp \
          -H "Content-Type: application/json" \
          -H "Accept: application/json, text/event-stream" \
          -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' 2>/dev/null \
          | grep -o '"name":"[^"]*"' | wc -l || echo "0")
        if [ "$TOOLS" -gt "0" ]; then
          echo "QMD MCP server started successfully with $TOOLS tools available"
          exit 0
        fi
        sleep 1
      done

      echo "QMD MCP server started but no tools available after 20 seconds"
      echo "Server logs:"
      cat /tmp/gh-aw/mcp-logs/qmd/server.log || true
      exit 1
---

<!--

## QMD MCP Server

Provides the QMD MCP server for on-device documentation search over the project docs,
agent definitions, and workflow authoring instructions.

QMD (Query Markup Documents) is a local search engine that combines BM25 full-text
search, vector semantic search, and LLM re-ranking — all running locally via
node-llama-cpp with GGUF models.

This shared configuration indexes three collections and starts a local HTTP
MCP server for the agent to query:

- `docs` — `docs/src/content/docs/` (documentation guides and reference)
- `agents` — `.github/agents/` (custom agent definitions and instructions)
- `aw` — `.github/aw/` (workflow authoring instructions and templates)

### Available Tools

- **`query`** — Search with typed sub-queries
  - Supports `lex` (BM25 keyword), `vec` (semantic), and `hyde` (LLM-expanded) query types
  - Combined via RRF + reranking for best quality
  - Parameters: `query` (string), `collections` (optional), `limit` (optional)

- **`get`** — Retrieve a specific document by path or docid
  - Supports fuzzy matching suggestions when path is not found
  - Parameters: `path` (string, e.g. `"docs/guides/getting-started.md"` or `"#abc123"`)

- **`multi_get`** — Batch retrieve by glob pattern or comma-separated list
  - Parameters: `pattern` (string, e.g. `"docs/reference/*.md"`)

- **`status`** — Index health and collection information
  - Returns collection names, document counts, and MCP server uptime

### Setup

Import this configuration in your workflow:

```yaml
imports:
  - shared/mcp/qmd-docs.md
```

### Example Usage

```yaml
---
on: workflow_dispatch
engine: copilot
imports:
  - shared/mcp/qmd-docs.md
---

# Documentation Search Workflow

Use the qmd tool to search the project documentation and answer questions.

1. Use `status` to see what collections are indexed
2. Use `query` with `lex` type for fast keyword search
3. Use `get` to retrieve specific documentation pages
```

### How It Works

The QMD index is pre-built by the `qmd-docs-indexer.yml` workflow on every trusted push
to `main` (path-filtered to the indexed directories) and on a daily schedule. This ensures
the index always reflects verified content.

At runtime (when this shared module is imported):

1. Node.js 24 is installed
2. QMD is installed globally from npm (`@tobilu/qmd`)
3. The pre-built qmd index is restored from `actions/cache` using a key derived from a hash of the docs, agents, and aw content
4. Collections are re-registered in the current session (`docs`, `agents`, `aw`) — required even after cache restore for qmd MCP tools to be served
5. The HTTP MCP server starts on `localhost:8181` (via `nohup`) and is verified to have tools available before the step completes

The `query` tool supports BM25 full-text search (`lex` type) out of the box.
For semantic vector search (`vec`/`hyde` types), run `qmd embed` before starting
the server to generate local GGUF model embeddings.

### Documentation

- **GitHub Repository**: https://github.com/tobi/qmd
- **npm Package**: https://www.npmjs.com/package/@tobilu/qmd
- **MCP Server docs**: https://github.com/tobi/qmd#mcp-server

-->
