---
name: Dev
on: 
  workflow_dispatch:
  push:
    branches:
      - serana*
engine: claude
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
---

Use serena to count lines of code in the repository.
Fail if serana mCP server is not available.