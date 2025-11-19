---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  close-issue:
    max: 2
    labels: [test-close]
timeout-minutes: 5
---

# Test Copilot Close Issue

Test the close-issue safe output functionality.

Close the following issues:
1. Issue #1 with a comment "Closing this test issue via automated workflow" and state_reason "completed"
2. Issue #2 with a comment "This issue is no longer relevant" and state_reason "not_planned"

Output as JSONL format with type "close_issue", including:
- issue_number
- comment
- state_reason
