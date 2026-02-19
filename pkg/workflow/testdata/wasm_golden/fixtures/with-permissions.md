---
name: with-permissions-test
description: Workflow with various permission configurations
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read
engine: copilot
timeout-minutes: 10
---

# Mission

Perform a comprehensive repository analysis with read permissions to contents, issues, pull requests, and actions.
