---
name: Test Actions Repo Override
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  create-issue:
    max: 1
---

# Test Actions Repo Override

When instructed, create an issue summarizing the repository state.
