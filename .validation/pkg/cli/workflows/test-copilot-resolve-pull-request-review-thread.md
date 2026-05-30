---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  resolve-pull-request-review-thread:
    max: 5
---

# Test Copilot Resolve Pull Request Review Thread

Test the `resolve_pull_request_review_thread` safe output type with the Copilot engine.

## Task

Resolve the pull request review thread with thread ID "PRRT_test123". This indicates the discussion in the thread has been addressed.

Output results in JSONL format using the `resolve_pull_request_review_thread` tool.
