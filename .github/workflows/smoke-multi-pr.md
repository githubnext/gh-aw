---
name: Smoke Multi PR
description: Test creating multiple pull requests in a single workflow run
on:
  schedule: every 24h
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke-multi-pr"]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  pull-requests: read
engine: copilot
strict: true
network:
  allowed:
    - defaults
    - node
tools:
  edit:
  bash:
    - "date"
    - "echo *"
safe-outputs:
  create-pull-request:
    title-prefix: "[smoke-multi-pr] "
    if-no-changes: "warn"
    labels: [ai-generated]
    expires: 2h
    max: 2
  add-comment:
    hide-older-comments: true
    max: 1
  messages:
    append-only-comments: true
    footer: "> 🧪 *Multi PR smoke test by [{workflow_name}]({run_url})*"
    run-started: "🧪 [{workflow_name}]({run_url}) is now testing multiple PR creation..."
    run-success: "✅ [{workflow_name}]({run_url}) successfully created multiple PRs."
    run-failure: "❌ [{workflow_name}]({run_url}) failed to create multiple PRs. Check the logs."
timeout-minutes: 10
---

# Smoke Test: Multiple Pull Request Creation

This workflow validates that multiple pull requests can be created in a single workflow run.

## Test Requirements

Create exactly TWO pull requests with distinct changes:

### PR 1: Documentation Update

1. Create a separate new branch off main `smoke-multi-pr-doc-${{ github.run_id }}-1` for the first PR
2. Create a file `tmp-smoke-multi-pr-doc-${{ github.run_id }}-1.txt` with content:
   ```
   Documentation smoke test for multi-PR workflow
   Run ID: ${{ github.run_id }}
   Created at: [current timestamp using date command]
   PR: 1 of 2
   ```
3. Create a pull request with:
   - Title: "PR 1: Documentation smoke test"
   - Body: "First of two PRs created by smoke-multi-pr workflow run ${{ github.run_id }}."

### PR 2: Configuration Update

1. Create a separate new branch off main `smoke-multi-pr-config-${{ github.run_id }}-2` for the second PR
2. Create a file `tmp-smoke-multi-pr-config-${{ github.run_id }}-2.txt` with content:
   ```
   Configuration smoke test for multi-PR workflow
   Run ID: ${{ github.run_id }}
   Created at: [current timestamp using date command]
   PR: 2 of 2
   ```
3. Create a pull request with:
   - Title: "PR 2: Configuration smoke test"
   - Body: "Second of two PRs created by smoke-multi-pr workflow run ${{ github.run_id }}."

## Success Criteria

Both PRs must be created successfully. After creating both PRs, add a comment to the triggering context summarizing:
- The two PR numbers created
- Links to both PRs
- Confirmation that multi-PR creation is working
