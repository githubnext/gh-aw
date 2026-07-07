---
on:
  workflow_dispatch:
permissions:
  pull-requests: read
engine: copilot
safe-outputs:
  create-pull-request-review-comment:
    max: 10
---

# Test Copilot Create Pull Request Review Comment

This is a test workflow to verify that Copilot can create review comments on pull requests.

Please add a review comment to the latest pull request saying "This is a test review comment from Copilot."