---
"gh-aw": patch
---

Fix Copilot MCP configuration tools field population

Updates the `renderGitHubCopilotMCPConfig` function to correctly populate the "tools" field in MCP configuration based on allowed tools from the configuration. Adds helper function `getGitHubAllowedTools` to extract allowed tools and defaults to `["*"]` when no allowed list is specified.
