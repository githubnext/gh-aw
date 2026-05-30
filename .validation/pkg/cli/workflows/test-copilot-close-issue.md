---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  close-issue:
    max: 1
---

# Test Copilot Close Issue

Test the `close_issue` safe output type with the Copilot engine.

## Task

Close issue #1 with a reason of "completed" and a comment "Closing this issue as it has been resolved by the automated test workflow."

Output results in JSONL format using the `close_issue` tool.
