---
on: workflow_dispatch
permissions:
  contents: read
  actions: write
engine: claude
safe-outputs:
  dispatch-workflow:
    max: 1
    allowed-workflows:
      - test-workflow-1.yml
      - test-workflow-2.yml
timeout-minutes: 5
---

# Test Dispatch Workflow

Test the dispatch-workflow safe output functionality.

Create a dispatch_workflow output to dispatch the workflow 'test-workflow-1.yml' with the following inputs:
- input1: "test-value-1"
- input2: "test-value-2"

Use the ref "main".

Output as JSONL format using the dispatch_workflow tool.
