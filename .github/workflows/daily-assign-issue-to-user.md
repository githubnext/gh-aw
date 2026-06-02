---
emoji: "🔧"
timeout-minutes: 10
strict: true
on:
  schedule: daily
  workflow_dispatch:
max-daily-effective-tokens: 100M
permissions:
  issues: read
  pull-requests: read
  contents: read
  copilot-requests: write
engine:
  id: copilot
  copilot-sdk: true
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [issues, pull_requests, repos]
safe-outputs:
  assign-to-user:
    target: "*"
  add-comment:
    target: "*"
imports:
  - shared/otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Auto-Assign Issue

Find ONE open issue that:
- **Has no assignees** - When you retrieve issues from GitHub, explicitly check the `assignees` field. Skip any issue where `issue.assignees` is not empty or has length > 0.
- Does not have label `ai-generated`
- Does not have a `campaign:*` label (these are managed by campaign orchestrators)
- Does not have labels: `no-bot`, `no-campaign`
- Was not opened by `github-actions` or any bot

Pick the oldest unassigned issue.

Then list the 5 most recent contributors from merged PRs. Pick one who seems relevant based on the issue type.

If you find a match:
1. Use `assign-to-user` to assign the issue
2. Use `add-comment` with a short explanation (1-2 sentences)

If no unassigned issue exists, exit successfully without taking action.

{{#runtime-import shared/noop-reminder.md}}