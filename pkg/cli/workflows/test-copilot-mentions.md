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
  mentions:
    enabled: true
    allowed-collaborators: true
    allow-context: true
    allowed:
      - copilot-bot
timeout-minutes: 5
---

# Test Copilot Mentions Config

Test the `mentions` safe-outputs configuration which controls @mention filtering
for all comment-producing safe output handlers.

Add a comment to this issue summarising the mentions policy:
- mentions are enabled
- repository collaborators are allowed
- context-based mentions are allowed
- explicitly allowed user: copilot-bot

Output as JSONL using the `add_comment` tool.
