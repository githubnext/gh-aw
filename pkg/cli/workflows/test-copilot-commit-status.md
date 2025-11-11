---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  commit-status:
    max: 1
    context: "test-workflow"
timeout-minutes: 5
---

# Test Commit Status

Test the commit-status safe output functionality.

Create a commit-status output with:
- state: "success"  
- description: "Test workflow completed successfully"

Output as JSONL format.
