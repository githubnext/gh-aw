---
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: custom
  steps:
    - name: Custom Step
      run: echo "Custom engine workflow"
timeout-minutes: 5
---

# Custom Engine Workflow

A workflow using a custom engine with defined steps.
