---
on:
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
engine: copilot
safe-outputs:
  close-discussion:
    max: 1
---

# Test Copilot Close Discussion

Test the `close_discussion` safe output type with the Copilot engine.

## Task

Close discussion #1 in this repository.

Output results in JSONL format using the `close_discussion` tool.
