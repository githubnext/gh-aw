---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  dispatch-workflow:
    max: 1
    workflows:
      - test-copilot-noop
---

# Test Copilot Dispatch Workflow

Test the `dispatch_workflow` safe output type with the Copilot engine.

## Task

Dispatch the workflow "test-copilot-noop" with no additional inputs.

Output results in JSONL format using the `dispatch_workflow` tool.
