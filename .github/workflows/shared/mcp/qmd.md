---
# QMD Documentation Search
# Local on-device search engine for the project documentation, agents, and workflow instructions
#
# Documentation: https://github.com/tobi/qmd
#
# Usage:
#   imports:
#     - shared/mcp/qmd.md

tools:
  qmd:
    runs-on: aw-gpu-runner-T4
    gpu: true
    checkouts:
      - name: gh-aw
        paths:
          - docs/src/content/docs/**
          - .github/agents/**
          - .github/aw/**
        context: "gh-aw project documentation, agent definitions, and workflow authoring instructions"

---

<qmd>
Use the `search` tool to find relevant documentation files with a natural language request — it queries a local vector database of project docs, agents, and workflow files. Read the returned file paths to get full content.

**Always use `search` first** when you need to find, verify, or search documentation:
- **Before using `find` or `bash` to list files** — use `search` to discover the most relevant docs for a topic
- **Before writing new content** — search first to check whether documentation already exists
- **When identifying relevant files** — use it to narrow down which documentation pages cover a feature or concept
- **When understanding a term or concept** — query to find authoritative documentation describing it

**Usage tips:**
- Use descriptive, natural language queries: e.g., `"how to configure MCP servers"` or `"safe-outputs create-pull-request options"` or `"permissions frontmatter field"`
- Always read the returned file paths to get the full content — `search` returns paths only, not content
- Combine multiple targeted queries rather than one broad query for better coverage
</qmd>
