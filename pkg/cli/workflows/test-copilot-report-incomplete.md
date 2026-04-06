---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  report-incomplete:
    max: 5
timeout-minutes: 5
---

# Test Copilot Report Incomplete

Test the `report_incomplete` safe output type with the Copilot engine.

## Task

Signal that the task could not be completed due to a simulated infrastructure failure. Report as incomplete with:
- **reason**: "Required MCP server was unavailable during workflow execution"
- **details**: "The workflow attempted to connect to the MCP server but received a connection refused error. This is a simulated failure for testing purposes."

Output results in JSONL format using the `report_incomplete` tool.
