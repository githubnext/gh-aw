---
on: 
  discussion:
    types: [created]
  discussion_comment:
    types: [created]
  workflow_dispatch:
    inputs:
      discussion_number:
        description: "Discussion number to add comment to"
        required: true
        type: string
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

Write a delightful poem about the last 3 pull requests and add it as a comment to discussion #${{ github.event.inputs.discussion_number || github.event.discussion.number }}.
