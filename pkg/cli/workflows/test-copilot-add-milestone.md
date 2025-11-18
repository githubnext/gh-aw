---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  add-milestone:
    allowed: ["v1.0", "v1.1", "v2.0"]
    max: 1
timeout-minutes: 5
---

# Test Copilot Add Milestone

Test the add-milestone safe output functionality with Copilot engine.

Add issue #1 to milestone "v2.0".

Output as JSONL format:
```
{"type": "add_milestone", "milestone": "v2.0", "item_number": 1}
```
