---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: codex
safe-outputs:
  add-milestone:
    allowed: ["v1.0", "v1.1", "v2.0"]
    max: 1
timeout-minutes: 5
---

# Test Codex Add Milestone

Test the add-milestone safe output functionality with Codex engine.

Add issue #1 to milestone "v1.1".

Output as JSONL format:
```
{"type": "add_milestone", "milestone": "v1.1", "item_number": 1}
```
