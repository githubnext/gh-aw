---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  assign-milestone:
    allowed: ["v1.0", "v1.1", "v2.0"]
    max: 1
timeout-minutes: 5
---

# Test Claude Assign Milestone

Test the assign-milestone safe output functionality with Claude engine.

Add issue #1 to milestone "v1.0".

Output as JSONL format:
```
{"type": "assign_milestone", "milestone": "v1.0", "item_number": 1}
```
