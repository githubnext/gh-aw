---
on:
  workflow_dispatch:
permissions:
  contents: read
  checks: write
engine: copilot
safe-outputs:
  create-check-run:
    max: 1
    name: "Copilot Analysis"
---

# Test Copilot Create Check Run

Test the `create_check_run` safe output type with the Copilot engine.

## Task

Create a GitHub Check Run with:
- **conclusion**: "success"
- **title**: "Copilot Analysis Complete"
- **summary**: "The automated analysis completed successfully. No issues were found."

Output results in JSONL format using the `create_check_run` tool.
