---
on:
  gateway-dispatch:
    repos: ["github/*"]
    slash-command:
      name: [review, "fix*"]
      events: [issue_comment, pull_request_comment]
    label-command:
      name: review
      events: [issues, pull_request]
      remove_label: true
permissions:
  contents: read
engine: copilot
---

# Gateway commands

Process the server-filtered command notifications in ${{ toJSON(gateway.events) }}.
