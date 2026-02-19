---
name: dispatch-with-inputs-test
description: Workflow with dispatch inputs
on:
  workflow_dispatch:
    inputs:
      query:
        description: "The search query"
        type: string
        required: true
      verbose:
        description: "Enable verbose output"
        type: boolean
        default: false
permissions:
  contents: read
engine: copilot
timeout-minutes: 10
---

# Mission

Process the search query provided via workflow dispatch inputs.
