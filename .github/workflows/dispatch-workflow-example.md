---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  staged: true  # Preview mode for safety
  dispatch-workflow:
    allowed-workflows:
      - "smoke-copilot.lock.yml"
timeout-minutes: 5
---

# Dispatch Workflow Example

This is a simple example demonstrating the dispatch-workflow safe output.

Create a dispatch_workflow output to trigger the smoke-copilot.lock.yml workflow with the following configuration:
- workflow: "smoke-copilot.lock.yml"
- ref: "main"

Output as JSONL format:
```
{"type": "dispatch_workflow", "workflow": "smoke-copilot.lock.yml", "ref": "main"}
```
