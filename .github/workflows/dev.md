---
on: 
  workflow_dispatch:
name: Dev
description: Create a poem about GitHub and save it to an issue
timeout-minutes: 5
strict: false
engine: copilot
agent-mode: dev

permissions: read-all

tools:
  github: false
  edit:
  bash: ["*"]
safe-outputs:
  create-issue:
  staged: true
---

Write a beautiful poem about GitHub and software development.
Create an issue with the poem using the create_issue tool.
