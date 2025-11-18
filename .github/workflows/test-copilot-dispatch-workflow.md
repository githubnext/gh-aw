---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  dispatch-workflow:
    max: 1
    allowed-workflows:
      - "smoke-copilot.lock.yml"
timeout-minutes: 5
---

# Test Dispatch Workflow (Copilot)

Test the dispatch-workflow safe output functionality using Copilot engine.

Create a dispatch_workflow output in JSONL format to trigger smoke-copilot.lock.yml with:
- ref: main
- inputs: { environment: "test", verbose: "true" }

Format as:
```
{"type": "dispatch_workflow", "workflow": "smoke-copilot.lock.yml", "ref": "main", "inputs": {"environment": "test", "verbose": "true"}}
```
