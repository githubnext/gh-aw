---
"gh-aw": patch
---

Document Claude `bypassPermissions` + `--allowed-tools` security boundary: clarify in AGENTS.md, engines reference, and MCP guide that `--allowed-tools` is silently ignored in `bypassPermissions` mode (unrestricted bash) and that the MCP gateway `allowed:` filter is the sole effective tool boundary in that case.
