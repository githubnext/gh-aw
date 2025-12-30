---
"gh-aw": patch
---

Auto-detect GitHub MCP lockdown based on repository visibility.

When the GitHub tool is enabled and `lockdown` is not specified, the
compiler inserts a detection step that sets `lockdown: true` for public
repositories and `false` for private/internal repositories. The detection
defaults to lockdown on API failure for safety.

