---
on: 
  workflow_dispatch:
  command:
    name: dev
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
tools:
  github:
---

Write a poem in 3 emojis about the last pull request and publish an issue.
