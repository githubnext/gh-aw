---
# QMD Documentation Search
# Local on-device vector search over documentation and other files.
#
# Documentation: https://github.com/tobi/qmd
#
# Usage:
#   imports:
#     - uses: shared/qmd.md
#       with:
#         checkouts:
#           - name: docs
#             pattern: "docs/**/*.md"
#             context: "Project documentation"
#         searches:
#           - name: issues
#             type: issues
#             max: 500
#             github-token: ${{ secrets.GITHUB_TOKEN }}
#         cache-key: "qmd-index-${{ hashFiles('docs/**') }}"
#         gpu: true
#         runs-on: aw-gpu-runner-T4

import-schema:
  checkouts:
    type: array
    required: false
    description: >
      List of documentation collections to index. Each item may include:
      name (string), pattern (glob string), ignore (string[]), context (string),
      checkout (object with repository/ref/path/token options).
  searches:
    type: array
    required: false
    description: >
      List of GitHub code or issue searches to add to the index. Each item may include:
      name (string), type ("code"|"issues"), query (string), min (number),
      max (number), github-token (string), github-app (object).
  cache-key:
    type: string
    required: false
    description: >
      GitHub Actions cache key for the search index.
      Omit for an ephemeral per-run index.
      Use cache-key alone (without checkouts or searches) for read-only mode.
  gpu:
    type: boolean
    required: false
    description: >
      Enable GPU acceleration for faster embedding generation. Default: false.
  runs-on:
    type: string
    required: false
    description: >
      Runner image for the indexing job. Default: ubuntu-latest.

tools:
  qmd:
    checkouts: ${{ github.aw.import-inputs.checkouts }}
    searches: ${{ github.aw.import-inputs.searches }}
    cache-key: ${{ github.aw.import-inputs.cache-key }}
    gpu: ${{ github.aw.import-inputs.gpu }}
    runs-on: ${{ github.aw.import-inputs.runs-on }}

---

<qmd>
Use the `search` tool to find relevant documentation and content with a natural language request — it queries a local vector database built from the configured collections.

**Always use `search` first** when you need to find, verify, or look up information:
- **Before using `find` or `bash` to list files** — use `search` to discover the most relevant content
- **Before writing new content** — search first to check whether similar content already exists
- **When identifying relevant files** — use it to narrow down which files cover a feature or concept
- **When understanding a term or concept** — query to find authoritative documentation

**Usage tips:**
- Use descriptive, natural language queries: e.g., `"how to configure MCP servers"` or `"safe-outputs create-pull-request options"` or `"permissions frontmatter field"`
- Always read the returned file paths to get the full content — `search` returns paths only, not content
- Combine multiple targeted queries rather than one broad query for better coverage
</qmd>
