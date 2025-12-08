---
on: 
  workflow_dispatch:
name: Dev
description: Create a poem about GitHub and save it to repo-memory
timeout-minutes: 5
strict: false
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github: false
imports:
  - shared/gh.md
safe-outputs:
  create-issue:
  staged: true
---

Read the last pull request using `githubissues-gh` tool
and create an issue with the summary.
