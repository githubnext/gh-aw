---
# Shared skip-if-match configuration for scheduled/periodic workflows.
# Prevents duplicate issues/PRs by skipping when an open item with the same title prefix exists.
#
# Usage:
#   imports:
#     - uses: shared/skip-if-issue-open.md
#       with:
#         title-prefix: "[my-workflow]"
#         kind: issue        # optional: issue|pr (default: issue)

import-schema:
  title-prefix:
    type: string
    required: true
    description: "Title prefix to search for (for example '[my-workflow]')"
  kind:
    type: string
    default: issue
    description: "Match open issue or pr"

on:
  skip-if-match: 'is:${{ github.aw.import-inputs.kind }} is:open in:title "${{ github.aw.import-inputs.title-prefix }}"'
---
