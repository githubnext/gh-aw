---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  update-pull-request:
    max: 1
---

# Test Copilot Update Pull Request

Test the `update_pull_request` safe output type with the Copilot engine.

## Task

Update pull request #1 with a new title "Updated PR Title" and body "This PR body was updated by the Copilot test workflow to verify the update_pull_request safe output type works correctly."

Output results in JSONL format using the `update_pull_request` tool.
