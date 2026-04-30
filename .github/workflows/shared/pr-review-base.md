---
# Base configuration for pull request code review workflows.
# Bundles: github-guard-policy + pr-code-review-config + standard PR review tools.
#
# Usage:
#   imports:
#     - uses: shared/pr-review-base.md
#       with:
#         min-integrity: approved   # optional

import-schema:
  min-integrity:
    type: string
    default: "approved"
    description: "Minimum integrity level required for tool access"

imports:
  - shared/github-guard-policy.md
  - shared/pr-code-review-config.md

permissions:
  contents: read
  pull-requests: read

tools:
  cli-proxy: true
  github:
    min-integrity: ${{ github.aw.import-inputs.min-integrity }}
    toolsets: [pull_requests, repos]

safe-outputs:
  noop:
---
