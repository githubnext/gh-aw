---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  create-issue:
    max: 1
  data: true
timeout-minutes: 5
---

# Test Copilot Data Mode

Test the `data` safe-outputs configuration which enables structured data mode
for body-based safe outputs. Here it is enabled for any object.

Create an issue summarising the data mode feature:
- title: "Data Mode Test"
- body: "This workflow validates that safe-outputs.data is enabled, allowing structured data to accompany body-based safe outputs."

Output as JSONL using the `create_issue` tool.
