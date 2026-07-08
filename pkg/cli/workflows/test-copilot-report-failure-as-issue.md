---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  noop:
    max: 1
  report-failure-as-issue: false
timeout-minutes: 5
---

# Test Copilot Report Failure As Issue

Test the `report-failure-as-issue` safe-outputs configuration which controls
whether workflow failures are automatically reported as GitHub issues.

Setting `report-failure-as-issue: false` disables automatic failure issue creation.

Use `noop` to confirm the configuration:
- message: "report-failure-as-issue is disabled for this workflow"

Output as JSONL using the `noop` tool.
