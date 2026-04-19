---
name: Smoke CI
description: Smoke CI workflow that comments via safeoutputs CLI without invoking an LLM
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
engine:
  id: copilot
  command: >-
    bash -lc 'if [ "${GITHUB_EVENT_NAME}" = "pull_request" ]; then safeoutputs
    add_comment --body "✅ smoke-ci: safeoutputs CLI comment only run (${GITHUB_RUN_ID})";
    else safeoutputs noop --message "smoke-ci: push event - no PR context, no action needed"; fi'
tools:
  mount-as-clis: true
safe-outputs:
  add-comment:
    hide-older-comments: true
    max: 1
  threat-detection: false
features:
  mcp-cli: true
timeout-minutes: 5
strict: true
---

Run exactly one CLI action: use the mounted `safeoutputs` CLI to add a short comment.
If there is no PR context, use `safeoutputs noop` with a brief message.
Do not call any LLM tools or perform any additional analysis.
