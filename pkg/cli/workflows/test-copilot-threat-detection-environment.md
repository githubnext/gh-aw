---
description: Test workflow for top-level environment propagation to threat detection
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task summary'
        required: true
        default: 'Check environment propagation'

environment: production
permissions: read-all

engine: copilot

safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automated]
    max: 1
  threat-detection: true

timeout-minutes: 10
---

# Test Threat Detection Environment Propagation

This workflow verifies that when a top-level `environment` is configured,
the compiled `detection` job inherits it.

Create an issue summarizing the provided task input.
