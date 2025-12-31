---
"gh-aw": patch
---

Fix safe-inputs MCP server start step so tool secrets are passed to the server

Safe-inputs tools with `env:` configuration were not receiving their secrets because the MCP server start step
exported variables that didn't exist in its environment. The start step now injects the collected tool secrets
via an `env:` block so the Node.js server process inherits the required secrets.

