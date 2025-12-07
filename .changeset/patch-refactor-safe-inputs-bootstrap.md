---
"gh-aw": patch
---

Refactor safe-inputs MCP server bootstrap to remove duplicated startup logic and centralize
config loading, tool handler resolution, and secure config cleanup.

Adds a shared `safe_inputs_bootstrap.cjs` module and updates stdio/HTTP transports to use it.

Fixes githubnext/gh-aw#5786

