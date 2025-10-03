---
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, critical, security]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
engine: claude
---

# Test Claude Labeled Names Filter

This workflow tests label name filtering for labeled/unlabeled events.

When a bug, critical, or security label is added or removed from an issue, analyze the label change and provide a brief status update in a comment.

Include in your comment:
- Which label was added or removed
- Whether this was a labeled or unlabeled action
- A brief note about the significance of this label
