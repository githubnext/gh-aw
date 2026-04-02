---
# Daily Audit Discussion Configuration
# Provides standardized safe-outputs config for workflows that publish daily audit discussions.
#
# Usage:
#   imports:
#     - uses: shared/daily-audit-discussion.md
#       with:
#         title-prefix: "[my-workflow] "
#         expires: 3d          # optional, default: 3d

import-schema:
  title-prefix:
    type: string
    required: true
    description: "Title prefix for created discussions, e.g. '[daily-report] '"
  expires:
    type: string
    default: "3d"
    description: "How long to keep discussions before expiry (e.g. 1d, 3d, 7d)"

safe-outputs:
  create-discussion:
    expires: ${{ github.aw.import-inputs.expires }}
    category: "audits"
    title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
    max: 1
    close-older-discussions: true
---
