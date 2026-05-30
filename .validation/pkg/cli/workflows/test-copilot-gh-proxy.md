---
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    mode: gh-proxy
---

# Test Copilot GH Proxy

Verify that `tools.github.mode: gh-proxy` uses CLI proxy guidance and does not register GitHub MCP.
