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

mcp-scripts:
  qmd-search:
    description: "Search project documentation and return ranked excerpts. Use for reading matching passages. Collections: docs, agents, aw."
    inputs:
      query:
        type: string
        required: true
        description: "Search query"
      collection:
        type: string
        required: false
        description: "Limit search to a collection: docs, agents, or aw"
      limit:
        type: number
        required: false
        default: 10
        description: "Maximum number of results to return"
    run: |
      set -e
      ARGS=(search "$INPUT_QUERY" --json -n "${INPUT_LIMIT:-10}")
      [[ -n "${INPUT_COLLECTION:-}" ]] && ARGS+=(--collection "$INPUT_COLLECTION")
      qmd "${ARGS[@]}"

  qmd-query:
    description: "Find relevant file paths in project documentation. Returns file paths and scores. Use to discover files before fetching with qmd-get."
    inputs:
      query:
        type: string
        required: true
        description: "Search query"
      collection:
        type: string
        required: false
        description: "Limit search to a collection: docs, agents, or aw"
      min_score:
        type: number
        required: false
        default: 0.4
        description: "Minimum relevance score threshold (0–1)"
    run: |
      set -e
      ARGS=(query "$INPUT_QUERY" --files --min-score "${INPUT_MIN_SCORE:-0.4}")
      [[ -n "${INPUT_COLLECTION:-}" ]] && ARGS+=(--collection "$INPUT_COLLECTION")
      qmd "${ARGS[@]}"

  qmd-get:
    description: "Retrieve the full content of a documentation file by path."
    inputs:
      path:
        type: string
        required: true
        description: "File path returned by qmd-query"
    run: |
      set -e
      qmd get "$INPUT_PATH"
---
