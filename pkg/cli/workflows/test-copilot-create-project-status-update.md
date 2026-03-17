---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  create-project-status-update:
    max: 1
---

# Test Copilot Create Project Status Update

Test the `create_project_status_update` safe output type with the Copilot engine.

## Task

Create a status update for a GitHub Project V2. Set the status to "ON_TRACK" with a body message "All tasks are progressing as planned. No blockers identified."

Output results in JSONL format using the `create_project_status_update` tool.
