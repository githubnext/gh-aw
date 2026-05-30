---
on:
  issues:
    types: [labeled]
    names: [priority-high, urgent, P0]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
engine: copilot
---

# Test Copilot Labeled Names Filter

This workflow tests label name filtering with Copilot for labeled events only.

When a priority-high, urgent, or P0 label is added to an issue, acknowledge the priority escalation with a comment.

Your comment should:
- Acknowledge the high-priority label that was added
- Suggest next steps for urgent handling
- Keep the response brief and actionable
