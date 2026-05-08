---
# Meta-Analysis Base - Standard tool stack for workflows that analyze other agentic workflows.
# Bundles: agentic-workflows tool + cli-proxy + GitHub MCP in gh-proxy mode.
#
# Usage:
#   imports:
#     - uses: shared/meta-analysis-base.md
#       with:
#         toolsets: [default, actions, repos]  # optional, default: [default]

import-schema:
  toolsets:
    type: array
    default: [default]
    description: "GitHub MCP toolsets to enable (e.g. [default, actions, repos])"

tools:
  cli-proxy: true
  agentic-workflows:
  github:
    mode: gh-proxy
    toolsets: ${{ github.aw.import-inputs.toolsets }}
---

**IMPORTANT**: When analyzing agentic workflows, use the `agentic-workflows` tool to read workflow files.
