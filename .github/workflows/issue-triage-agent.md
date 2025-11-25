---
on:
  schedule:
    - cron: "0 14 * * 1-5"
  workflow_dispatch:
permissions:
  issues: read
engine: copilot
tools:
  github:
    read-only: true
    toolsets: [issues, labels]
safe-outputs:
  add-labels:
    allowed: [bug, feature, enhancement, documentation, question, help-wanted, good-first-issue]
---

# Issue Triage Agent

List open issues in ${{ github.repository }} that have no labels. For each unlabeled issue, analyze the title and body, then add one of the allowed labels: `bug`, `feature`, `enhancement`, `documentation`, `question`, `help-wanted`, or `good-first-issue`. Skip issues that already have labels.
