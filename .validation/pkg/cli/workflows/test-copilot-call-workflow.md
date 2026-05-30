---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  call-workflow:
    max: 1
    workflows:
      - test-copilot-noop
---

# Test Copilot Call Workflow

Test the `call_workflow` safe output type with the Copilot engine.

## Task

Call the reusable workflow "test-copilot-noop" as a fan-out job.

Output results in JSONL format using the `call_workflow` tool.
