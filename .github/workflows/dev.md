---
on: 
  workflow_dispatch:
name: Dev
description: Lock the last issue with copilot-no-firewall label
timeout-minutes: 5
strict: false
engine: copilot

permissions: read-all

tools:
  github:
    toolsets: [issues]
  edit:
  bash: ["*"]
safe-outputs:
  lock-issue:
    target: "*"
    labels: [smoke-copilot-no-firewall]
    max: 5
steps:
  - name: Download issues data
    run: |
      gh issue list --limit 10 --json number,title,body,labels,state --label copilot-no-firewall
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

Lock the last open issue that has the "smoke-copilot-no-firewall" label.

Find the most recent open issue with the "smoke-copilot-no-firewall" label and lock it with:
- A comment explaining why it's being locked: "Locking this issue as it has the copilot-no-firewall label"
- Lock reason: "resolved"
