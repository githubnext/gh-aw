---
"gh-aw": patch
---

Expand the "default" GitHub MCP toolset into individual, action-friendly
toolsets (exclude `users`) and add support for the `action-friendly`
keyword. This ensures generated workflows expand `default` into the
`context,repos,issues,pull_requests` toolsets which are compatible with
GitHub Actions tokens.

