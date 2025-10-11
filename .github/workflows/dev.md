---
on: 
  discussion:
    types: [created]
  discussion_comment:
    types: [created]
name: Dev
engine: codex
permissions:
  contents: read
  actions: read
  discussions: write
tools:
  github:
    mode: "remote"
    toolset:
      - "pull_requests"
safe-outputs:
  staged: true
  add-comment:
---

Write a delightful poem about the last 3 pull requests and add it as a comment to the discussion.
