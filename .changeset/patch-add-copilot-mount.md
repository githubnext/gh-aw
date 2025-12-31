---
"gh-aw": patch
---

Add mount for `/home/runner/.copilot` so the Copilot CLI inside the AWF container can
access its MCP configuration. This fixes smoke-test failures where MCP tools were
unavailable (playwright, safeinputs, github).

Fixes: githubnext/gh-aw#8157

