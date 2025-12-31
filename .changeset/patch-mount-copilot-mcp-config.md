---
"gh-aw": patch
---

Mount Copilot MCP config directory into AWF container so Copilot-based workflows can access MCP servers.

This exposes the Copilot config directory at `/home/runner/.copilot` to the AWF container with read-write
permissions, allowing the Copilot CLI to read and write MCP configuration and runtime state.

Fixes: githubnext/gh-aw#8157

