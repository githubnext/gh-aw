---
on:
  workflow_dispatch:
permissions: read-all
engine: copilot
safe-outputs:
  staged: true
  create-issue:
    title-prefix: "[staged] "
    max: 1
---

# Test Staged Create Issue

Verify that `staged: true` works together with the `create-issue` handler.

Create an issue titled "Staged test issue" with the body "This issue was created in staged mode."
