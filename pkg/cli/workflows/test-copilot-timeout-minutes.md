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
  timeout-minutes: 30
timeout-minutes: 5
---

# Test Copilot Safe-Outputs Timeout Minutes

Test the `safe-outputs.timeout-minutes` configuration which overrides the
default 45-minute timeout for the safe_outputs job.

Use `noop` to report that the timeout-minutes is configured to 30 minutes:
- message: "safe-outputs job timeout set to 30 minutes"

Output as JSONL using the `noop` tool.
