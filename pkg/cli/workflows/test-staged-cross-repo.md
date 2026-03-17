---
on:
  workflow_dispatch:
permissions: read-all
engine: copilot
safe-outputs:
  staged: true
  create-issue:
    title-prefix: "[cross-repo staged] "
    max: 1
    target-repo: org/other-repo
---

# Test Staged Create Issue Cross-Repo

Verify that `staged: true` is emitted even when a `target-repo` is configured.
`staged` mode is independent of the `target-repo` setting.

Create an issue in the target repository titled "Cross-repo staged test issue."
