---
on:
  command:
    name: help-bot
permissions:
  issues: write
engine: copilot
tools:
  github:
    toolsets: [default]
timeout-minutes: 5
---

# Help Bot Command

Respond to /help-bot mentions with helpful information.
