---
name: Smoke CI
description: Smoke CI workflow that exercises pull request safe outputs through an agent session
on:
  push:
    branches: [main]
  pull_request:
    types: [opened, synchronize, reopened]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
safe-outputs:
  add-comment:
    hide-older-comments: true
    max: 1
  add-labels:
    max: 1
    allowed: [ai-generated]
  remove-labels:
    max: 1
    allowed: [ai-generated]
  threat-detection: false
features:
  mcp-cli: true
timeout-minutes: 5
strict: true
---

For pull_request events, call the safe output tools in this exact order:
1. `add_comment` with a short smoke-ci message that includes the run URL.
2. `add_labels` with exactly `["ai-generated"]`.
3. `remove_labels` with exactly `["ai-generated"]`.

For non-pull_request events, call `noop` with a short message indicating no PR context.

Do not run any shell commands.
Do not call any tools other than `add_comment`, `add_labels`, `remove_labels`, or `noop`.
