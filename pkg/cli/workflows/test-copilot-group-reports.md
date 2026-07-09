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
  group-reports: true
timeout-minutes: 5
---

# Test Copilot Group Reports

Test the `group-reports` safe-outputs configuration which, when true, creates
a parent "Failed runs" issue to group agent failure reports (default: false).

Create an issue summarising the group-reports feature:
- title: "Group Reports Test"
- body: "This workflow validates that group-reports is enabled. When agent failures occur, they are grouped under a parent tracking issue."

Output as JSONL using the `create_issue` tool.
