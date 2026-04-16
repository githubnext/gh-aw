---
# Daily Audit Base - Standard stack for daily audit workflows publishing to GitHub Discussions.
# Bundles: daily-audit-discussion + reporting guidelines + OTLP observability.
#
# Usage:
#   imports:
#     - uses: shared/daily-audit-base.md
#       with:
#         title-prefix: "[my-workflow] "
#         expires: "1d"      # optional, default: 3d

import-schema:
  title-prefix:
    type: string
    required: true
    description: "Title prefix for created discussions, e.g. '[daily-report] '"
  expires:
    type: string
    default: "3d"
    description: "How long to keep discussions before expiry"

imports:
  - uses: shared/daily-audit-discussion.md
    with:
      title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
      expires: "${{ github.aw.import-inputs.expires }}"
  - shared/reporting.md
  - shared/observability-otlp.md
---
