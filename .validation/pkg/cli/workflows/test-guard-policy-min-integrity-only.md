---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    min-integrity: none
---

# Test Guard Policy with min-integrity Only

This workflow verifies that specifying only `min-integrity` under `tools.github`
works correctly without requiring an explicit `repos` field.

When `repos` is omitted, it should default to `all`, allowing the workflow to compile
successfully.

Please list the first 3 open issues in this repository.
