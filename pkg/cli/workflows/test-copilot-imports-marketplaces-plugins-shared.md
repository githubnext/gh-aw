---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  aw:
    - shared/marketplace-plugins.md
  plugins:
    - main-plugin
---

# Test Copilot Imports Plugins with Shared Import

This workflow tests that `imports.plugins` values from a shared
agentic workflow (imported via `imports.aw`) are merged with the main workflow's own values.

Process the issue and respond with a helpful comment.
