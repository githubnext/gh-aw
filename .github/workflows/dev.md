---
on: 
  workflow_dispatch:
name: Dev
engine: codex
permissions:
  contents: read
  actions: read
tools:
  github:
    mode: "remote"
    toolset:
      - "pull_requests"
safe-outputs:
  staged: true
  create-issue:
    title-prefix: "[dev] "
    labels: [dev, sub-task, poetry]
---

Write a poem about the last 3 pull requests and publish an issue.
