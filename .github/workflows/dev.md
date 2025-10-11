---
on: 
  workflow_dispatch:
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
    target: "*"
---

Write a delightful poem about the last 3 pull requests and add it as a comment to discussion #1.
