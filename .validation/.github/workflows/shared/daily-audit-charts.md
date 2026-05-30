---
# Daily Audit Charts - Standard stack for daily audit workflows with trend charts.
# Bundles: daily-audit-base + trending-charts-simple.
#
# Usage:
#   imports:
#     - uses: shared/daily-audit-charts.md
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
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "${{ github.aw.import-inputs.title-prefix }}"
      expires: "${{ github.aw.import-inputs.expires }}"
  - shared/trending-charts-simple.md
---
