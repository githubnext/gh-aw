---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  actions:
    add-smoked-label:
      uses: actions-ecosystem/action-add-labels@v1
      description: Add the 'smoked' label to the current pull request
      env:
        GITHUB_TOKEN: ${{ github.token }}
---

# Test Safe Output Actions

This workflow demonstrates `safe-outputs.actions`, which mounts a GitHub Action
as a once-callable MCP tool.

When done, call `add_smoked_label` with `{"labels": "smoked"}` to add the label
to the current pull request.
