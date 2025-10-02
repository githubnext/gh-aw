---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  add-labels:
    allowed: [bug, enhancement, documentation]
    target: "*"
---

# Test Add Labels with Target

This workflow demonstrates the `target` field for `add-labels`.

With `target: "*"`, the workflow can add labels to any issue by specifying
the `issue_number` in the output.

Please add the label "bug" to issue #1 and the label "documentation" to issue #2.
