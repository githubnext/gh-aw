---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  marketplaces:
    - https://marketplace.example.com
  plugins:
    - my-plugin
---

# Test Copilot Imports Marketplaces and Plugins

This workflow tests that `imports.marketplaces` and `imports.plugins` are compiled into
`copilot plugin marketplace add` and `copilot plugin install` setup steps before the agent runs.

Process the issue and respond with a helpful comment.
