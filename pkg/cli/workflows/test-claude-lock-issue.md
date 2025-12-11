---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
  issues: write
engine: claude
safe-outputs:
  lock-issue:
    target: triggering
    max: 1
timeout-minutes: 5
strict: false
---

# Test Lock Issue - Claude

Test the lock-issue safe output functionality with Claude engine.

## Task

Create a lock_issue output to lock the current issue that triggered this workflow.

1. Add a comment explaining: "Locking this issue as the discussion has become unproductive and off-topic."
2. Set the lock reason to "off-topic"
3. Output as JSONL format with type "lock_issue"

The lock-issue safe output should:
- Lock the issue that triggered this workflow
- Add the comment before locking
- Apply the off-topic lock reason

Example JSONL output:
```jsonl
{"type":"lock_issue","body":"Locking this issue as the discussion has become unproductive and off-topic.","lock_reason":"off-topic"}
```
