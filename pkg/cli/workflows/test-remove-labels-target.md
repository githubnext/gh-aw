---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  remove-labels:
    allowed: [bug, enhancement, documentation]
    target: "*"
timeout_minutes: 5
---

# Test Remove Labels with Target

This workflow demonstrates the `target` field for `remove-labels`.

With `target: "*"`, the workflow can remove labels from any issue by specifying
the `issue_number` in the output.

Please remove the label "bug" from issue #1 and the label "documentation" from issue #2.
