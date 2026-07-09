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
  max-bot-mentions: "5"
timeout-minutes: 5
---

# Test Copilot Max Bot Mentions

Test the `max-bot-mentions` safe-outputs configuration which limits the maximum
number of bot trigger references (e.g. `fixes #123`) allowed before filtering
(default: 10). Supports integer or GitHub Actions expression.

Add a comment summarising the max-bot-mentions policy:
- message: "This workflow has max-bot-mentions set to 5. References beyond this threshold are filtered to prevent runaway bot loops."

Output as JSONL using the `add_comment` tool.
