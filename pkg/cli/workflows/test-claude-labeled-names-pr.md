---
on:
  pull_request:
    types: [labeled, unlabeled]
    names: [ready-for-review, needs-changes, approved]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
engine: claude
---

# Test Claude PR Labeled Names Filter

This workflow tests label name filtering for pull request labeled/unlabeled events.

When a ready-for-review, needs-changes, or approved label is added or removed from a PR, provide a brief status comment.

Include:
- The label that changed
- Whether it was added or removed
- Appropriate next steps based on the label state
