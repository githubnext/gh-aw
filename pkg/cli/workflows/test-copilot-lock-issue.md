---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
  issues: write
engine: copilot
tools:
  github:
    toolsets: [default]
safe-outputs:
  lock-issue:
    target: "*"
    required-labels: "spam,lock-candidate"
    max: 2
timeout-minutes: 5
strict: false
---

# Test Lock Issue - Copilot

Test the lock-issue safe output functionality with Copilot engine and label filtering.

## Task

Create lock_issue outputs to lock issues that have both "spam" and "lock-candidate" labels.

1. Lock issue #10 with comment "Locking this issue due to spam" and reason "spam"
2. Lock issue #20 with comment "Locking as discussion is off-topic" and reason "off-topic"
3. Output as JSONL format with type "lock_issue"

The lock-issue safe output should:
- Only lock issues that have BOTH "spam" AND "lock-candidate" labels (configured via required-labels filter)
- Lock specific issues by number (target is configured as "*")
- Add appropriate comments before locking
- Apply the specified lock reasons

Example JSONL outputs:
```jsonl
{"type":"lock_issue","issue_number":10,"body":"Locking this issue due to spam","lock_reason":"spam"}
{"type":"lock_issue","issue_number":20,"body":"Locking as discussion is off-topic","lock_reason":"off-topic"}
```
