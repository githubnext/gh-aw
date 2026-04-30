---
# Bundle for daily automated code improvement workflows that create pull requests.
# Bundles: activation-app + reporting guidelines + standardized create-pull-request safe-outputs.
#
# Usage:
#   imports:
#     - uses: shared/daily-pr-base.md
#       with:
#         title-prefix: "[my-workflow] "
#         expires: "1d"
#         labels: [automation]
#         reviewers: [copilot]

import-schema:
  title-prefix:
    type: string
    required: true
    description: "Title prefix for created PRs, e.g. '[my-workflow] '"
  expires:
    type: string
    default: "1d"
    description: "How long to keep open PRs before expiry"
  labels:
    type: array
    default: [automation]
    description: "Labels to apply to created PRs"
  reviewers:
    type: array
    default: [copilot]
    description: "Reviewers to assign to created PRs"

imports:
  - shared/activation-app.md
  - shared/reporting.md

safe-outputs:
  create-pull-request:
    expires: ${{ github.aw.import-inputs.expires }}
    title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
    labels: ${{ github.aw.import-inputs.labels }}
    reviewers: ${{ github.aw.import-inputs.reviewers }}
  noop:
---
