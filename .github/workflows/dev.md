---
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
  contents: read
  actions: read
tools:
  github:
    toolsets: [default]
timeout-minutes: 10
---

# Dev

## Rule Definition

Say hello to the world.