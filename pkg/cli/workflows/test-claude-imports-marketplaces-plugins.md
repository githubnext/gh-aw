---
on: issues
permissions:
  contents: read
  issues: read
engine: claude
imports:
  plugins:
    - my-plugin
---

# Test Claude Imports Plugins

This workflow tests that `imports.plugins` is compiled into
`claude plugin install` setup steps before the agent runs.

Process the issue and respond with a helpful comment.
