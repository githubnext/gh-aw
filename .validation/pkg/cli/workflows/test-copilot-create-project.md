---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  create-project:
    max: 1
---

# Test Copilot Create Project

Test the `create_project` safe output type with the Copilot engine.

## Task

Create a new GitHub Project V2 with the title "Test Project from Copilot" and description "This project was created automatically by the Copilot test workflow to verify the create_project safe output type works correctly."

Output results in JSONL format using the `create_project` tool.
