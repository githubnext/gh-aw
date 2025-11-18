---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  dispatch-workflow:
    max: 3
    allowed-workflows:
      - "smoke-claude.lock.yml"
      - "smoke-codex.lock.yml"
      - "smoke-copilot.lock.yml"
timeout-minutes: 5
---

# Test Dispatch Workflow (Claude)

Test the dispatch-workflow safe output functionality using Claude engine.

Create dispatch_workflow outputs in JSONL format to trigger the following workflows:
1. smoke-claude.lock.yml
2. smoke-codex.lock.yml (with ref: main)
3. smoke-copilot.lock.yml (with inputs: { test: "true" })

Format each output as:
```
{"type": "dispatch_workflow", "workflow": "workflow-file.yml"}
```
