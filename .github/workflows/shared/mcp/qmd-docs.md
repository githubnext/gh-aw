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
    uses: actions/cache/restore@v5.0.4
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
      qmd vsearch "$INPUT_QUERY" --files --min-score "${INPUT_MIN_SCORE:-0.4}"

---

<qmd>
Use `qmd-query` to find relevant documentation files with a natural language request — it queries a local vector database of project docs, agents, and workflow files. Read the returned file paths to get full content.

**Always use `qmd-query` first** when you need to find, verify, or search documentation:
- **Before using `find` or `bash` to list files** — use `qmd-query` to discover the most relevant docs for a topic
- **Before writing new content** — search first to check whether documentation already exists
- **When identifying relevant files** — use it to narrow down which documentation pages cover a feature or concept
- **When understanding a term or concept** — query to find authoritative documentation describing it

**Usage tips:**
- Use descriptive, natural language queries: e.g., `"how to configure MCP servers"` or `"safe-outputs create-pull-request options"` or `"permissions frontmatter field"`
- Lower `min_score` (e.g., `0.3`) to get broader results; raise it (e.g., `0.6`) to get only the most closely matching files
- Always read the returned file paths to get the full content — `qmd-query` returns paths only, not content
- Combine multiple targeted queries rather than one broad query for better coverage
</qmd>
