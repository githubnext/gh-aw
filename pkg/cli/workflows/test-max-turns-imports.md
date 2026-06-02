---
on:
  workflow_dispatch:
imports:
  - ./shared/max-turns-import.md
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
    allowed: [issue_read]
---

# Shared max-turns import fixture

Verifies that top-level `max-turns` from a shared workflow import is preserved through CLI compilation.
