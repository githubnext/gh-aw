---
on: 
  workflow_dispatch:
name: Dev
timeout-minutes: 5
strict: false
engine: copilot
permissions:
  contents: read
  issues: read
tools:
  github: false
imports:
  - shared/gh.md
---

Read the last issue using the `safeissues-gh` tool and print its title.