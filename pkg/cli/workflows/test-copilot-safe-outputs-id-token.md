---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  noop:
    max: 1
  id-token: write
timeout-minutes: 5
---

# Test Copilot Safe-Outputs ID Token

Test the `safe-outputs.id-token` configuration which overrides the `id-token`
permission for the safe_outputs job. Setting `id-token: write` force-adds the
permission; `id-token: none` disables it.

Use `noop` to confirm the id-token configuration:
- message: "safe-outputs id-token permission set to write"

Output as JSONL using the `noop` tool.
