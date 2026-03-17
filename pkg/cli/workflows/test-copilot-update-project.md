---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  update-project:
    max: 5
---

# Test Copilot Update Project

Test the `update_project` safe output type with the Copilot engine.

## Task

Add issue #1 to a GitHub Project V2. Set the status field to "In Progress" for the added item.

Output results in JSONL format using the `update_project` tool.
