# Test Status Comment Decoupling

This workflow tests the decoupling of status-comment from ai-reaction.

---
name: test-status-comment-decoupling
on:
  reaction: eyes
  status-comment: false
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  create-issues:
    max: 1
---

Test workflow that uses ai-reaction without status comments.
This should:
1. Add 👀 reaction in pre-activation job
2. NOT add "Add comment with workflow run link" step in activation job
3. NOT add "Update reaction comment" step in conclusion job
