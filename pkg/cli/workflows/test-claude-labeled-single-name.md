---
on:
  issues:
    types: [labeled]
    names: documentation
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
engine: claude
---

# Test Claude Single Label Name Filter

This workflow tests label name filtering with a single label name (string format instead of array).

When the documentation label is added to an issue, provide guidance on documentation best practices.

Your comment should:
- Acknowledge the documentation label
- Provide 2-3 tips for good documentation
- Keep it concise and helpful
