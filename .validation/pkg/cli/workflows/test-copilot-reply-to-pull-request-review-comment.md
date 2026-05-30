---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  reply-to-pull-request-review-comment:
    max: 1
---

# Test Copilot Reply to Pull Request Review Comment

Test the `reply_to_pull_request_review_comment` safe output type with the Copilot engine.

## Task

Reply to pull request review comment #1 with the body "Thank you for the review comment. This is an automated test reply from the Copilot test workflow."

Output results in JSONL format using the `reply_to_pull_request_review_comment` tool.
