---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  remove-labels:
    allowed: [bug, enhancement, documentation]
    max: 3
timeout_minutes: 5
---

# Test Remove Labels - Copilot

Test the remove-labels safe output type with the Copilot engine.

Remove the labels "bug" and "enhancement" from the current issue.

Output as JSONL format.
