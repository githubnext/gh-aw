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

<qmd>
Use the `qmd` CLI to search project documentation. Three collections are available: `docs`, `agents`, `aw`.

Commands:
- `qmd search "<query>" --json -n 10` — ranked excerpts (read content)
- `qmd query "<query>" --files --min-score 0.4` — matching file paths only
- `qmd search "<query>" --collection docs --json` — search a specific collection
- `qmd get <path>` — retrieve a full document
- `qmd multi-get "docs/reference/*.md"` — batch retrieve by glob
- `qmd status` — index health and collection info

Use `search` to read matching passages; use `query` to discover relevant files before fetching them with `get`.
</qmd>
