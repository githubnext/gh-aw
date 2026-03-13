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
      qmd collection add "${GITHUB_WORKSPACE}" --name gh-aw --glob "docs/src/content/docs/**,.github/agents/**,.github/aw/**" 2>/dev/null || true

mcp-scripts:
  qmd-query:
    description: "Find relevant file paths in project documentation using vector similarity search. Returns file paths and scores."
    inputs:
      query:
        type: string
        required: true
        description: "Natural language search query"
      min_score:
        type: number
        required: false
        default: 0.4
        description: "Minimum relevance score threshold (0–1)"
    run: |
      set -e
      qmd query "$INPUT_QUERY" --files --min-score "${INPUT_MIN_SCORE:-0.4}"

---

<qmd>
Use `qmd-query` to find relevant documentation files with a natural language request — it queries a local vector database of project docs, agents, and workflow files. Read the returned file paths to get full content.
</qmd>
