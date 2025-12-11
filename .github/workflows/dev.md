---
on: 
  workflow_dispatch:
name: Dev
description: Create an empty pull request for agent to push changes to
timeout-minutes: 5
strict: false
engine: copilot

permissions: read-all

tools:
  github: false
  edit:
  bash: ["*"]
imports:
  - shared/gh.md
safe-outputs:
  create-pull-request:
    allow-empty: true
  staged: true
steps:
  - name: Download issues data
    run: |
      gh pr list --limit 1 --json number,title,body,author,createdAt,mergedAt,state,url
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

Create an empty pull request that prepares a branch for future changes.
The pull request should have:
- Title: "Feature: Prepare branch for agent updates"
- Body: "This is an empty pull request created to prepare a feature branch that an agent can push changes to later."
- Branch name: "feature/agent-updates"
