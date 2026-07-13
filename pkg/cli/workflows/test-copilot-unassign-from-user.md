---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  unassign-from-user:
    max: 1
---

# Test Copilot Unassign From User

Test the `unassign_from_user` safe output type with the Copilot engine.

## Task

Unassign user "octocat" from issue #1 in this repository.

Output results in JSONL format using the `unassign_from_user` tool.
