---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
engine: copilot
safe-outputs:
  add-comment:
    max: 1
  urls: allowed-or-code-region
timeout-minutes: 5
---

# Test Copilot Safe-Outputs URLs Policy

Test the `safe-outputs.urls` configuration which controls URL sanitization
policy. Valid values:
- `allowed-only` (default): redact URLs not in the allowed-domains list
- `allowed-or-code-region`: also allow URLs inside code blocks/regions

Add a comment summarising the URL policy:
- message: "This workflow uses the 'allowed-or-code-region' URL policy, which permits URLs inside code blocks even if not in the allowed-domains list."

Output as JSONL using the `add_comment` tool.
