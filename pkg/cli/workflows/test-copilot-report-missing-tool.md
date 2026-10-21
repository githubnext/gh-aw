---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  report-missing-tool:
    max: 1
timeout-minutes: 5
---

# Test Report Missing Tool

Pretend a tool needed to complete the task is unavailable, then report it using the `missing_tool` safe-output tool.

Output as JSONL format using the `missing_tool` tool.
