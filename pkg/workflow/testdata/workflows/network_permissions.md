---
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.github.com"
    - "*.example.com"
tools:
  web-fetch:
  github:
    toolsets: [default]
timeout-minutes: 10
---

# Network Permissions Workflow

A workflow with custom network permissions configured.
