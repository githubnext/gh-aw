---
name: "Playground: Org project update issue"
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

imports:
  - shared/safe-output-app.md

safe-outputs:
  update-project: {}

---

# Issue Updater

Goal: prove we can **update a Project item** that points to a real GitHub Issue.

Project board: <https://github.com/orgs/githubnext/projects/66>

Task: Update all issue items to Status "In Progress".
