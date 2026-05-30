---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  submit-pull-request-review:
    max: 1
---

# Test Copilot Submit Pull Request Review

Test the `submit_pull_request_review` safe output type with the Copilot engine.

## Task

Submit a COMMENT review on pull request #1 with the body "This is a test review comment submitted by the automated test workflow to verify the submit_pull_request_review safe output type works correctly."

Output results in JSONL format using the `submit_pull_request_review` tool.
