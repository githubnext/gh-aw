---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: codex
safe-outputs:
  dispatch-workflow:
    max: 2
    allowed-workflows:
      - "smoke-codex.lock.yml"
      - "smoke-claude.lock.yml"
timeout-minutes: 5
---

# Test Dispatch Workflow (Codex)

Test the dispatch-workflow safe output functionality using Codex engine.

Create dispatch_workflow outputs in JSONL format to trigger:
1. smoke-codex.lock.yml (no ref or inputs)
2. smoke-claude.lock.yml (with ref: main)

Format each output as:
```
{"type": "dispatch_workflow", "workflow": "workflow-file.yml"}
```
