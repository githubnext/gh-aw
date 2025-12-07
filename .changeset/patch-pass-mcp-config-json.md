---
"gh-aw": patch
---

Pass MCP config as a JSON string to Copilot's `--additional-mcp-config` instead
of writing the config to a file or relying on the `GH_AW_MCP_CONFIG` file-path
environment variable. The implementation now generates the MCP config into a
shell variable `GH_AW_MCP_CONFIG_JSON` (via a heredoc) and passes its contents
directly to the CLI. This removes file I/O and avoids needing to manage a temp
file for Copilot's MCP configuration.

Fixes githubnext/gh-aw#2417

