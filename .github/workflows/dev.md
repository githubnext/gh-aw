---
name: Dev
description: Update issues on an org-owned Project Board
on:
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.TEST_ORG_PROJECT_WRITE }} # fine-grained PAT with scopes Organizational: `Projects: Read & Write` and `Metadata: Read` and `Issues: Read & Write`

safe-outputs:
  update-project:
    github-token: ${{ secrets.TEST_ORG_PROJECT_WRITE }}
---

# Issue Updater

Goal: prove we can **update a Project item** that points to a real GitHub Issue.

Project board: <https://github.com/orgs/githubnext/projects/66>

Task: Update all issue items that are currently on the project board with Status "In Progress".
