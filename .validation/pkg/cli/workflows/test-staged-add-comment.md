---
on:
  workflow_dispatch:
permissions: read-all
engine: copilot
safe-outputs:
  staged: true
  add-comment:
    max: 1
---

# Test Staged Add Comment

Verify that `staged: true` works together with the `add-comment` handler.

Add a comment to issue #1 saying "Staged test comment."
