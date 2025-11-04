---
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
permissions:
  contents: read
  pull-requests: read
  issues: read
name: Smoke Copilot
engine: copilot
network:
  firewall: true
tools:
  edit:
  bash:
  github:
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
---

Review the last 5 merged pull requests in this repository and post summary in an issue.