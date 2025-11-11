---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  commit-status:
    context: "test-workflow"
timeout-minutes: 5
---

# Test Commit Status

Test the commit-status safe output functionality.

Create a commit-status output with:
- state: "success"
- description: "Test workflow completed successfully"

Output as JSONL format.
