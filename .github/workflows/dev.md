---
on: 
  workflow_dispatch:
name: Dev
description: Create a poem about GitHub and save it to repo-memory
timeout-minutes: 5
strict: false
engine: copilot
agent-mode: dev

permissions: read-all

tools:
  github: false
  edit:
  bash: ["*"]
imports:
  - shared/gh.md
safe-outputs:
  create-issue:
  staged: true
steps:
  - name: Download issues data
    run: |
      gh pr list --limit 1 --json number,title,body,author,createdAt,mergedAt,state,url
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

Read the last pull request using `githubissues-gh` tool
and create an issue with the summary.
