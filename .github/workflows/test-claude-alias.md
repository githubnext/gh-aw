---
on:
  alias:
    name: test-claude-alias
  reaction: eyes

engine: 
  id: claude

safe-outputs:
  add-issue-comment:
---

Add a reply comment to issue #${{ github.event.issue.number }} answering the question "${{ needs.task.outputs.text }}" given the context of the repo, starting with saying you're Claude. If there is no alias write out a haiku about the repo.

