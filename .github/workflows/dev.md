---
name: Dev
on: 
  workflow_dispatch:
  push:
    branches:
      - serana*
engine: codex
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
imports:
  - shared/serena-mcp.md
---

Use serena to count lines of code in the repository.
Fail if serena mCP server is not available.