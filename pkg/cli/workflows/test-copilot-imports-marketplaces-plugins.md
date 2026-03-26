---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  plugins:
    - my-plugin
---

# Test Copilot Imports Plugins

This workflow tests that `imports.plugins` is compiled into
`copilot plugin install` setup steps before the agent runs.

Process the issue and respond with a helpful comment.
