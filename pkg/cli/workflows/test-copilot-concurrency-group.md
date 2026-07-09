---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  noop:
    max: 1
  concurrency-group: test-safe-outputs-${{ github.ref }}
timeout-minutes: 5
---

# Test Copilot Concurrency Group

Test the `safe-outputs.concurrency-group` configuration which sets a GitHub
Actions concurrency group for the safe_outputs job (cancel-in-progress is
always false for safe_outputs).

Use `noop` to confirm the configuration:
- message: "safe-outputs concurrency group is configured"

Output as JSONL using the `noop` tool.
