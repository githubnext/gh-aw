---
# Bundle for daily/scheduled code quality workflows that create GitHub issues.
# Bundles: activation-app + reporting guidelines + standardized create-issue safe-outputs.
#
# Usage:
#   imports:
#     - uses: shared/daily-issue-base.md
#       with:
#         title-prefix: "[my-workflow] "
#         expires: "2d"      # optional, default: 2d
#         labels: [automation, cookie]
#         assignees: [copilot]  # optional, default: []
#         max: 1               # optional, default: 1
#         close-older-issues: false # optional, default: false

import-schema:
  title-prefix:
    type: string
    required: true
    description: "Title prefix for created issues, e.g. '[my-workflow] '"
  expires:
    type: string
    default: "2d"
    description: "How long to keep issues before expiry"
  labels:
    type: array
    default: [automated-analysis, cookie]
    description: "Labels to apply to created issues"
  assignees:
    type: array
    default: []
    description: "Assignees for created issues"
  max:
    type: number
    default: 1
    description: "Maximum issues allowed per run"
  close-older-issues:
    type: boolean
    default: false
    description: "Close previous issues from the same workflow key"

imports:
  - shared/activation-app.md
  - shared/reporting.md

safe-outputs:
  create-issue:
    expires: ${{ github.aw.import-inputs.expires }}
    title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
    labels: ${{ github.aw.import-inputs.labels }}
    assignees: ${{ github.aw.import-inputs.assignees }}
    max: ${{ github.aw.import-inputs.max }}
    close-older-issues: ${{ github.aw.import-inputs.close-older-issues }}
  noop:
---
