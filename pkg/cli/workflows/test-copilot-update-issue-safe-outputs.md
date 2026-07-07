---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  update-issue:
    max: 5
    status: true
    title: true
    body: true
---

# Test Copilot Update Issue Safe Outputs

Test the `update_issue` safe output type with the Copilot engine.

## Task

Update issue #1: set the title to "Updated by automated test workflow" and append the text "Updated by test." to the body.

Output results in JSONL format using the `update_issue` tool.
