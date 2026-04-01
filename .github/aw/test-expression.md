---
on:
  workflow_call:
    inputs:
      engine-version:
        type: string
engine:
  id: copilot
  version: ${{ inputs.engine-version }}
---
Fix the bug
