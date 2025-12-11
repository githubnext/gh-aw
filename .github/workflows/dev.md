---
on: 
  workflow_dispatch:
name: Dev
description: Test create-project and update-project safe outputs
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
  create-project:
    max: 1
  update-project:
    max: 5
  create-issue:
  staged: true
steps:
  - name: Download issues data
    run: |
      gh pr list --limit 1 --json number,title,body,author,createdAt,mergedAt,state,url
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

Create a new GitHub Projects v2 board and then add items to it.

1. Use the `create-project` safe output to create a project board named "Dev Project Test" linked to this repository.
2. Confirm the project exists (idempotent: re-using the same name should return the existing board).
3. Use the `update-project` safe output to add at least one issue from this repository to the "Dev Project Test" project.
4. Set simple fields on the project item such as Status (e.g., "Todo") and Priority (e.g., "Medium").
5. If any step fails, explain what happened and how to fix it in a short summary.
