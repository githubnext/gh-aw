---
"gh-aw": patch
---

Remove `docker.io/mcp/brave-search` container from workflows due to Critical and High CVEs with no upstream fix available (issue #48546).

The `shared/mcp/brave.md` MCP server definition has been emptied, its import removed from `brave.md` and `mcp-inspector.md`, and the pinned digest removed from `actions-lock.json`. Re-enable by restoring the `mcp-servers` block in `shared/mcp/brave.md` once a patched image is published upstream.
