---
"gh-aw": patch
---

Refactor patch generation from workflow step to MCP server

Moves git patch generation from a dedicated workflow step to the safe-outputs MCP server, where it executes when `create_pull_request` or `push_to_pull_request_branch` tools are called. This provides immediate error feedback when no changes exist, rather than discovering it later in processing jobs.
