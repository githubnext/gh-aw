---
name: Smoke CI
description: Smoke CI workflow that exercises pull request safe outputs through an agent session
on:
  push:
    branches: [main]
  schedule: daily
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
  cache-memory: true
  repo-memory:
    branch-name: memory/smoke-ci
    description: "Smoke CI persisted repo-memory entries"
    file-glob:
      - "*.md"
  github:
safe-outputs:
  create-issue:
    max: 1
    title-prefix: "[smoke-ci] "
    labels: [ai-generated]
    close-older-issues: true
    close-older-key: "smoke-ci-memory-safe-outputs"
  add-comment:
    hide-older-comments: true
    max: 1
  add-labels:
    max: 1
    allowed: [ai-generated]
  remove-labels:
    max: 1
    allowed: [ai-generated]
  update-issue:
    body:
    max: 1
    target: "*"
  update-pull-request:
    body: true
    max: 1
    target: "*"
  threat-detection: false
features:
  mcp-cli: true
timeout-minutes: 5
strict: true
---

For all events, call the tools in this exact order:
1. Use `cache-memory` to write a short JSON record for this run.
2. Use `repo-memory` to write a short markdown run note.
3. Use `create_issue` with temporary ID `aw_smokeci` and include the run URL in the body.
4. Use `update_issue` targeting `aw_smokeci` with `operation: "append"` to add a second line confirming the update succeeded.

For pull_request events, then call these safe output tools in this exact order:
1. `add_comment` with a short smoke-ci message that includes the run URL.
2. `add_labels` with exactly `["ai-generated"]`.
3. `remove_labels` with exactly `["ai-generated"]`.
4. `update_pull_request` on the triggering pull request with `operation: "append"` and a short body line including the run URL.

For scheduled runs (non-pull_request), use GitHub MCP to find the newest open pull request in `${{ github.repository }}`:
- If one exists, call `update_pull_request` for that PR number with `operation: "append"` and a short body line including the run URL.
- If none exists, call `noop` with a short message indicating no PR was available.

Do not run any shell commands.
Do not call any tools other than `cache-memory`, `repo-memory`, `github`, `create_issue`, `update_issue`, `add_comment`, `add_labels`, `remove_labels`, `update_pull_request`, or `noop`.
