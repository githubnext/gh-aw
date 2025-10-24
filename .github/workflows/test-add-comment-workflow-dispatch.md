---
on:
  workflow_dispatch:
name: Test Add Comment on Workflow Dispatch
engine: copilot
tools:
  github:
safe-outputs:
  staged: true
  add-comment:
    max: 1
timeout_minutes: 5
---

# Test Add Comment from Workflow Dispatch

This workflow tests the new functionality that allows add-comment to work from workflow_dispatch events.

When triggered by workflow_dispatch (which is not a commentable event like issue or PR), the workflow should:
1. Resolve the current branch from GITHUB_REF or GITHUB_HEAD_REF
2. Search for the first open pull request matching that branch
3. If found, add a comment to that pull request
4. If not found, exit gracefully without error

Please analyze the current branch and create a brief comment summarizing:
- The branch name
- The current workflow run ID
- A confirmation that the add-comment feature is working for non-commentable events

Example comment format:
```
✅ Add-comment feature test successful!

- Branch: `feature-branch-name`
- Workflow Run: #12345
- Event: workflow_dispatch
- Status: Successfully found and commented on PR from non-commentable event
```
