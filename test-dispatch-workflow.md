---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  dispatch-workflow:
    allowed-workflows:
      - ci.yml
      - build.yml
timeout_minutes: 10
---

# Workflow Dispatcher

This workflow can dispatch other workflows based on certain conditions.

Use the dispatch-workflow tool to trigger the CI or build workflows when needed.
