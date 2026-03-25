---
on: issues
permissions:
  contents: read
  issues: read
engine: claude
imports:
  marketplaces:
    - https://marketplace.example.com
  plugins:
    - my-plugin
---

# Test Claude Imports Marketplaces and Plugins

This workflow tests that `imports.marketplaces` and `imports.plugins` are compiled into
`claude plugin marketplace add` and `claude plugin install` setup steps before the agent runs.

Process the issue and respond with a helpful comment.
