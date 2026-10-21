---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  report-missing-data:
    max: 1
timeout-minutes: 5
---

# Test Report Missing Data

Pretend required input data is unavailable, then report it using the `missing_data` safe-output tool.

Output as JSONL format using the `missing_data` tool.
