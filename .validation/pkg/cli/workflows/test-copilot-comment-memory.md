---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  comment-memory:
    max: 1
    memory-id: test-memory
timeout-minutes: 5
---

# Test Copilot Comment Memory

Test the `comment_memory` safe output type with the Copilot engine.

## Task

Update or create a memory comment on issue #1 with the body "Memory update: this is a test of the comment_memory safe output type. Timestamp: now."

Output results in JSONL format using the `comment_memory` tool.
