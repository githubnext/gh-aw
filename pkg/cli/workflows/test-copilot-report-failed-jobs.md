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
  report-failed-jobs: false
timeout-minutes: 5
---

# Test Copilot Report Failed Jobs

Test the `report-failed-jobs` safe-outputs configuration which controls
whether failed non-builtin jobs are reported as issues (default: true). Here
it is disabled.

Create an issue summarising the report-failed-jobs feature:
- title: "Report Failed Jobs Test"
- body: "This workflow validates that report-failed-jobs is set to false, disabling automatic issue creation for failed non-builtin jobs."

Output as JSONL using the `create_issue` tool.
