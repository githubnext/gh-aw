---
on:
  workflow_dispatch:
permissions:
  contents: write
  pull-requests: write
engine: copilot
safe-outputs:
  create-pull-request:
    max: 1
  max-patch-files: 50
timeout-minutes: 5
---

# Test Copilot Max Patch Files

Test the `max-patch-files` safe-outputs configuration which limits the maximum
number of unique files allowed per `create-pull-request` patch (default: 100).

Create a pull request with a single file change and a body that notes the
max-patch-files limit is set to 50.

Output as JSONL using the `create_pull_request` tool.
