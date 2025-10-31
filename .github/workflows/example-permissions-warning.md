---
on: 
  workflow_dispatch:
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [repos, issues, pull_requests]
    read-only: false
---

# Example: Under-provisioned Permissions Warning

This workflow demonstrates the new warning behavior for under-provisioned permissions.

When compiled in non-strict mode (default), this workflow will produce a warning because:
- The `repos` toolset requires `contents: write` (but we only have `read`)
- The `issues` toolset requires `issues: write` (but we only have `read`)
- The `pull_requests` toolset requires `pull-requests: write` (but we don't have it at all)

The warning will suggest two options:
1. Add the missing write permissions
2. Or reduce the toolsets to only those that work with read-only permissions

In strict mode (with --strict flag), this would fail with an error instead.
