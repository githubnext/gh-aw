---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  link-sub-issue:
    max: 5
---

# Test Copilot Link Sub-Issue

Test the `link_sub_issue` safe output type with the Copilot engine.

## Task

Link issue #2 as a sub-issue of issue #1. This establishes a parent-child relationship between the two issues.

Output results in JSONL format using the `link_sub_issue` tool.
