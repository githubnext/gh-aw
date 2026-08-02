---
"gh-aw": patch
---

Remove `docker.io/mcp/markitdown` container from workflows due to Critical and High CVEs with no upstream fix available (issue #49515).

The `shared/mcp/markitdown.md` MCP server definition has been emptied, its import removed from `scout.md`, `pdf-summary.md`, and `mcp-inspector.md`, and the pinned digest removed from `actions-lock.json`. Re-enable by restoring the `mcp-servers` block in `shared/mcp/markitdown.md` and updating the pinned digest once a patched image is published upstream.
