---
"gh-aw": patch
---

Update GitHub Copilot CLI to version 0.0.340 and implement ${} syntax for MCP environment variables

This update upgrades the GitHub Copilot CLI from version 0.0.339 to 0.0.340 and implements the breaking change for MCP server environment variable configuration. The safe-outputs MCP server now uses the new `${VAR}` syntax for environment variable references instead of direct variable names.
