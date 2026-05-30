---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-discussion:
    max: 1
    category: general
---

# Test Copilot Create Discussion

Test the `create_discussion` safe output type with the Copilot engine.

## Task

Create a new GitHub discussion with the title "Test Discussion from Copilot" and the body "This discussion was created automatically by the Copilot test workflow to verify the create_discussion safe output type works correctly."

Output results in JSONL format using the `create_discussion` tool.
