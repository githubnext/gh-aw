---
"gh-aw": patch
---

Document `push_to_pull_request_branch` as append-only (force-push not supported) and auto-linearize merge commits before signed-commit push.

Previously, an agent that ran `git merge origin/main` (instead of `git rebase`) would produce a merge commit that `pushSignedCommits` unconditionally rejected, leaving the request completely unserviced.

Changes:
- Updated the MCP tool description to explicitly state that `push_to_pull_request_branch` is **append-only** — force-push is NOT supported — and that merge commits must be avoided by using `git rebase` instead of `git merge`.
- Updated the agent prompt guidance (`safe_outputs_push_to_pr_branch.md`) with the same constraints.
- Added auto-linearization in `push_to_pull_request_branch.cjs`: when the commit range contains merge commits and signed commits are required, the handler now squashes the range into a single regular commit (preserving the file-level outcome) and emits a warning so authors know to prefer `git rebase` in future runs.
