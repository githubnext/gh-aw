---
"gh-aw": patch
---

Upgrade `gh-aw-mcpg` to `v0.2.2`, updating the default MCP gateway image version and recompiling workflow lock files. This includes a fix for more reliable search result extraction where MCP Gateway now falls back to URL-based parsing when extracting issue/PR numbers and repository names from search results.
