---
on: push
permissions:
  contents: read
  issues: write
safe-outputs:
  create-issue:
  staged: true
---

# Test Staged Workflow

This workflow should create an issue in staged mode (preview only).