---
engine:
  id: copilot
  display-name: GitHub Copilot CLI
  description: Uses GitHub Copilot CLI with MCP server support
  runtime-id: copilot
  provider:
    name: github
  auth:
    - role: api-key
      secret: COPILOT_GITHUB_TOKEN
---

<!-- # GitHub Copilot CLI

Shared engine configuration for GitHub Copilot CLI. -->
