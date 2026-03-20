---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
    staged: true
    title-prefix: "[staged] "
    max: 1
  add-labels:
    max: 3
---

# Test Staged Permissions (Per-Handler)

Verify that when only specific handlers have `staged: true`, the compiled
safe_outputs job only includes permissions required by the non-staged handlers.

Here `create-issue` is staged (no write permissions for it), and `add-labels`
is not staged (needs `issues: write` and `pull-requests: write`).
