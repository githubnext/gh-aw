---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "example.com"
---

# Test Workflow

Test without explicit firewall config.
