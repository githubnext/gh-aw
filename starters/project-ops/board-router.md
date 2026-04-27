---
name: Issue Board Router
description: Add newly opened issues to a GitHub Project board and set the Priority field based on labels.
on:
  issues:
    types: [opened, labeled]
permissions:
  contents: read
  issues: read
safe-outputs:
  update-project:
    project: https://github.com/orgs/YOUR-ORG/projects/1
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    max: 1
---

Add the triggering issue to the project board defined above.
Set the Priority field to `High` if the issue has a `priority-high` or `critical` label, `Low` if it has `good first issue`, and `Medium` for everything else.
If the issue has a `bug` label also set the Status field to `In Triage`.
