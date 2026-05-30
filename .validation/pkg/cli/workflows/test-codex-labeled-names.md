---
on:
  issues:
    types: [labeled, unlabeled]
    names: [enhancement, feature, needs-review]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
engine: codex
---

# Test Codex Labeled Names Filter

This workflow tests label name filtering with Codex for labeled/unlabeled events.

When an enhancement, feature, or needs-review label is added or removed, provide feedback on the label change.

Comment should mention:
- The label that triggered this workflow
- The action type (labeled or unlabeled)
- A short suggestion related to this label change
