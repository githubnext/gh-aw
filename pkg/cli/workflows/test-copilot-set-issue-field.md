---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  set-issue-field:
    max: 5
timeout-minutes: 5
---

# Test Copilot Set Issue Field

Test the `set_issue_field` safe output type with the Copilot engine.

## Task

Set a custom field on issue #1 in the current repository.

Use the following parameters:
- **issue_number**: 1
- **field_name**: "Status"
- **value**: "In Progress"

Output results in JSONL format using the `set_issue_field` tool.
