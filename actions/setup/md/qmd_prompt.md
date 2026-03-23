<qmd>
Use the qmd search tool to find relevant documentation files using vector similarity — it queries a local index built from the configured documentation globs. Read the returned file paths to get full content.

**Always use the qmd search tool first** when you need to find, verify, or search documentation:
- **Before using `find` or `bash` to list files** — use qmd search to discover the most relevant docs for a topic
- **Before writing new content** — search first to check whether documentation already exists
- **When identifying relevant files** — use it to narrow down which documentation pages cover a feature or concept
- **When understanding a term or concept** — query to find authoritative documentation describing it

**Usage tips:**
- Use descriptive, natural language queries: e.g., `"how to configure MCP servers"` or `"safe-outputs create-pull-request options"` or `"permissions frontmatter field"`
- Always read the returned file paths to get the full content — the qmd search tool returns paths only, not content
- Combine multiple targeted queries rather than one broad query for better coverage
- A lower score threshold gives broader results; a higher one (e.g., `0.6`) returns only the most closely matching files
</qmd>
