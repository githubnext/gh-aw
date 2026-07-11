---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  assign-to-user:
    max: 1
---

# Test Copilot Assign To User

Test the `assign_to_user` safe output type with the Copilot engine.

## Task

Assign issue #1 to user "octocat" in this repository.

Output results in JSONL format using the `assign_to_user` tool.
