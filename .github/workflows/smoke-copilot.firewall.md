---
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
name: Smoke Copilot Firewall
engine: copilot
features:
  firewall: true
tools:
  github:
safe-outputs:
    staged: true
    create-issue:
      min: 1
timeout_minutes: 10
strict: true
---

Review the last 5 merged pull requests in this repository and post summary in an issue.
