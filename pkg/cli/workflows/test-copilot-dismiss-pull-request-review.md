---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  dismiss-pull-request-review:
    max: 10
---

# Test Copilot Dismiss Pull Request Review

Test the `dismiss_pull_request_review` safe output type with the Copilot engine.

## Task

Dismiss the most recent pending review on pull request #1 with the message "Dismissed by automated test workflow."

Output results in JSONL format using the `dismiss_pull_request_review` tool.
