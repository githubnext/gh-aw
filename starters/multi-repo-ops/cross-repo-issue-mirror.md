---
name: Cross-Repo Issue Mirror
description: Mirror issues labeled "upstream" to a companion repository for cross-team tracking.
on:
  label_command: upstream
permissions:
  contents: read
  issues: read
safe-outputs:
  github-token: ${{ secrets.GH_AW_CROSS_REPO_PAT }}
  create-issue:
    target-repo: "YOUR-ORG/YOUR-UPSTREAM-REPO"
    title-prefix: "[mirror] "
    labels: [upstream-mirror]
    max: 1
---

Read the triggering issue's title, body, and labels, then create a mirror issue in the upstream repository.
Include a link back to the original issue and a one-sentence summary of the problem.
Do not include any personally identifiable information from the issue author.
