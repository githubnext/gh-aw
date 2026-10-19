---
on:
  bots: [cao-gateway]
  gateway-dispatch:
    repos: ["github/*"]
    orgs: [github]
    issue:
      types: [opened, reopened]
      filters:
        labels: triage
    pull-request:
      types: [ready_for_review, closed]
      filters:
        base: main
        draft: false
    discussion:
    issue-comment:
      types: [created, edited, deleted]
    pull-request-comment:
    pull-request-review:
      types: [submitted]
      filters:
        state: approved
    pull-request-review-comment:
    pull-request-review-thread:
    discussion-comment:
    commit-comment:
permissions:
  contents: read
engine: copilot
---

# Gateway GitHub notifications

Process the minimal notifications in ${{ toJSON(gateway.events) }}.
Retrieve additional accessible resource details with authenticated GitHub tools.
