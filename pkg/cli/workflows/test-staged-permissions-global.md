---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  staged: true
  create-issue:
    title-prefix: "[staged] "
    max: 1
  add-labels:
    max: 3
  create-discussion:
    max: 1
---

# Test Staged Permissions (Global)

Verify that when `staged: true` is set globally, the compiled safe_outputs job
has **no** job-level `permissions:` block (all handlers are staged, so no write
permissions are needed).
