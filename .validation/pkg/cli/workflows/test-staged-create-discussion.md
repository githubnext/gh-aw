---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  staged: true
  create-discussion:
    max: 1
    category: general
---

# Test Staged Create Discussion

Verify that `staged: true` works together with the `create-discussion` handler.

Create a discussion titled "Staged test discussion" with the body "This discussion was created in staged mode."
