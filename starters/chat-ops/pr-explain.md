---
name: PR Explainer
description: Explain pull request changes in plain English when someone types /explain in a PR comment.
on:
  slash_command:
    name: explain
    events: [pull_request_comment]
permissions:
  contents: read
  pull-requests: read
safe-outputs:
  add-comment:
    max: 1
---

Explain the pull request changes in plain English for a non-technical stakeholder.
Read the diff and describe what changed, why it matters, and any notable risks in three bullet points.
Keep the explanation under 150 words.
