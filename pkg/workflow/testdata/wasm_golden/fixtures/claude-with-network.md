---
name: claude-with-network-test
description: Claude workflow with network configuration
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
timeout-minutes: 15
network:
  allowed:
    - defaults
    - node
    - python
---

# Mission

Install dependencies and analyze the project.
