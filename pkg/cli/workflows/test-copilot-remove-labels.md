---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  remove-labels:
    max: 5
---

# Test Copilot Remove Labels

Test the `remove_labels` safe output type with the Copilot engine.

## Task

Remove the label "bug" from issue #1.

Output results in JSONL format using the `remove_labels` tool.
