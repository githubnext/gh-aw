---
# QMD Documentation Search
# Local on-device search engine for the project documentation, agents, and workflow instructions
#
# Documentation: https://github.com/tobi/qmd
#
# Usage:
#   imports:
#     - shared/mcp/qmd-docs.md

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
  - name: Register QMD collections
    run: |
      DOCS_DIR="${GITHUB_WORKSPACE}/docs/src/content/docs"
      AGENTS_DIR="${GITHUB_WORKSPACE}/.github/agents"
      AW_DIR="${GITHUB_WORKSPACE}/.github/aw"

      [ -d "$DOCS_DIR" ]   && qmd collection add "$DOCS_DIR"   --name docs   2>/dev/null || true
      [ -d "$AGENTS_DIR" ] && qmd collection add "$AGENTS_DIR" --name agents 2>/dev/null || true
      [ -d "$AW_DIR" ]     && qmd collection add "$AW_DIR"     --name aw     2>/dev/null || true
---

<!--

## QMD Documentation Search

Provides local documentation search over the project docs, agent definitions, and
workflow authoring instructions using the `qmd` CLI.

QMD (Query Markup Documents) is a local search engine that combines BM25 full-text
search, vector semantic search, and LLM re-ranking — all running locally via
node-llama-cpp with GGUF models.

Three collections are indexed:

- `docs` — `docs/src/content/docs/` (documentation guides and reference)
- `agents` — `.github/agents/` (custom agent definitions and instructions)
- `aw` — `.github/aw/` (workflow authoring instructions and templates)

### Querying the Docs

Use the `qmd` CLI directly in workflow steps to search documentation.

**`qmd search`** — Returns a ranked list of matching document excerpts. Use this for
content-level search when you want to read the matching passages.

**`qmd query`** — Returns file paths and scores without full content. Use this to
discover which files are relevant before deciding which ones to retrieve with `qmd get`.

```bash
# Get structured JSON results for an LLM (document excerpts)
qmd search "authentication" --json -n 10

# List all relevant files above a relevance threshold (paths + scores only)
qmd query "error handling" --all --files --min-score 0.4

# Search a specific collection
qmd search "compile workflow" --collection docs --json

# Retrieve a specific document by path
qmd get docs/guides/getting-started.md

# Retrieve multiple documents matching a glob pattern
qmd multi-get "docs/reference/*.md"

# Show index health and collection info
qmd status
```

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

Search the project documentation and answer questions.

1. Run `qmd search "your topic" --json -n 10` to find relevant documents and read their content
2. Run `qmd query "your topic" --files --min-score 0.4` to list relevant file paths
3. Run `qmd get <path>` to retrieve the full content of a specific page
4. Use `--collection docs`, `--collection agents`, or `--collection aw` to narrow results
```

### Using CLI Output in Workflow Steps

Capture `qmd` output in a step and pass it to subsequent processing:

```bash
# Capture JSON search results and parse with jq
RESULTS=$(qmd search "authentication" --json -n 5)
echo "$RESULTS" | jq '.[].path'

# Save relevant file list to a variable for later retrieval
FILES=$(qmd query "error handling" --files --min-score 0.4)
for f in $FILES; do
  qmd get "$f"
done
```

### How It Works

The QMD index is pre-built by the `qmd-docs-indexer.yml` workflow on every trusted push
to `main` (path-filtered to the indexed directories) and on a daily schedule. This ensures
the index always reflects verified content.

At runtime (when this shared module is imported):

1. Node.js 24 is installed
2. QMD is installed globally from npm (`@tobilu/qmd`)
3. The pre-built qmd index is restored from `actions/cache` using a key derived from a hash of the docs, agents, and aw content
4. Collections are re-registered in the current session (`docs`, `agents`, `aw`)

The `search` command supports BM25 full-text search out of the box.
For semantic vector search, run `qmd embed` first to generate local GGUF model embeddings.

### Documentation

- **GitHub Repository**: https://github.com/tobi/qmd
- **npm Package**: https://www.npmjs.com/package/@tobilu/qmd

-->
