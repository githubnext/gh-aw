---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
  issues: write
engine: codex
safe-outputs:
  lock-issue:
    target: "*"
    max: 3
timeout-minutes: 5
strict: false
---

# Test Lock Issue - Codex

Test the lock-issue safe output functionality with Codex engine.

## Task

Create lock_issue outputs to lock specific issues by number.

1. Lock issue #100 with comment "Issue resolved - locking to prevent further discussion" and reason "resolved"
2. Lock issue #200 with comment "Locking due to spam activity" and reason "spam"
3. Lock issue #300 with comment "Discussion has become too heated" and reason "too heated"
4. Output as JSONL format with type "lock_issue"

The lock-issue safe output should:
- Lock specific issues by number (target is configured as "*")
- Add appropriate comments before locking
- Apply the specified lock reasons

Example JSONL outputs:
```jsonl
{"type":"lock_issue","issue_number":100,"body":"Issue resolved - locking to prevent further discussion","lock_reason":"resolved"}
{"type":"lock_issue","issue_number":200,"body":"Locking due to spam activity","lock_reason":"spam"}
{"type":"lock_issue","issue_number":300,"body":"Discussion has become too heated","lock_reason":"too heated"}
```
