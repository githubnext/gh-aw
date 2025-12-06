---
"gh-aw": patch
---

Move `get_me` out of the default GitHub MCP toolsets and into the `users` toolset.

Workflows that rely on the `get_me` tool must now opt in by adding `toolsets: [users]` under the `github` tools configuration.

This change updates the toolset mappings and documentation; tests were adjusted to ensure `get_me` is not available by default.

