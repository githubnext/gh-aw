---
# Standard Repo-Memory Configuration
# Provides a standardized repo-memory setup for workflows that need historical data persistence.
#
# Usage:
#   imports:
#     - uses: shared/repo-memory-standard.md
#       with:
#         branch-name: "memory/my-workflow"
#         description: "Historical my-workflow analysis results"

import-schema:
  branch-name:
    type: string
    required: true
    description: "Branch name for repo-memory storage (e.g. memory/my-workflow)"
  description:
    type: string
    required: true
    description: "Human-readable description of what is stored"
  max-file-size:
    type: integer
    default: 102400
    description: "Max file size in bytes (default: 100KB)"

tools:
  repo-memory:
    branch-name: ${{ github.aw.import-inputs.branch-name }}
    description: ${{ github.aw.import-inputs.description }}
    file-glob:
      - "${{ github.aw.import-inputs.branch-name }}/*.json"
      - "${{ github.aw.import-inputs.branch-name }}/*.jsonl"
      - "${{ github.aw.import-inputs.branch-name }}/*.csv"
      - "${{ github.aw.import-inputs.branch-name }}/*.md"
    max-file-size: ${{ github.aw.import-inputs.max-file-size }}
---
