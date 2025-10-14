---
"gh-aw": patch
---

Update q.md workflow to use MCP server tools instead of CLI commands

The Q agentic workflow was incorrectly referencing the `gh aw` CLI command, which won't work because the agent doesn't have GitHub token access. Updated all references to explicitly use the gh-aw MCP server's `compile` tool instead.
