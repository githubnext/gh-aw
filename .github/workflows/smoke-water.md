---
name: Smoke Water
description: Triggers smoke-workflow-call agentic workflow when the 'water' label is applied to a pull request
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["water"]
jobs:
  call-smoke-workflow-call:
    if: github.event_name == 'workflow_dispatch' || github.event.label.name == 'water'
    uses: ./.github/workflows/smoke-workflow-call.lock.yml
    secrets: inherit
    permissions:
      contents: read
      discussions: write
      issues: write
      pull-requests: write
---

This workflow is a dispatcher for the `smoke-workflow-call` agentic workflow.
It is triggered when the `water` label is applied to a pull request, or via `workflow_dispatch`.

No action is needed from the agent — the `call-smoke-workflow-call` job handles everything.

```json
{"noop": {"message": "Dispatcher workflow: smoke-workflow-call job handles all validation."}}
```
