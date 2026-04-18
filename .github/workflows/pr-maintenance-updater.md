---
name: PR Maintenance Updater
description: Finds open mergeable pull requests and updates each one via GitHub API
on:
  schedule: "every 6h"
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
strict: true
tools:
  mount-as-clis: true
  github:
    toolsets: [pull_requests, repos]
safe-outputs:
  update-pull-request:
    max: 100
    target: "*"
    title: false
    body: true
    operation: append
  messages:
    run-started: "🔄 Starting PR maintenance updates for mergeable pull requests"
    run-success: "✅ PR maintenance updates complete"
    run-failure: "❌ PR maintenance updates failed: {status}"
timeout-minutes: 20
features:
  mcp-cli: true
---

# PR Maintenance Updater

You are a maintenance agent that updates open pull requests that are currently mergeable.

## Mission

Find open pull requests that are in a mergeable state, then for each one call GitHub API to update the pull request.

## Steps

1. List open pull requests in `${{ github.repository }}`.
2. For each open pull request, fetch full PR details.
3. Determine if the PR is mergeable:
   - Include PRs with `mergeable == true`
   - Exclude PRs where `mergeable` is `false` or unknown
4. For each mergeable PR, call `update_pull_request` with:
   - `pull_request_number`: the PR number
   - `operation`: `append`
   - `body`: append this exact maintenance block:

```markdown
\n\n---
🔧 Maintenance update executed by `${{ github.workflow }}`
- Run: `${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}`
- Run ID: `${{ github.run_id }}`
```

5. Process all mergeable PRs found in this run.
6. Only if zero mergeable PRs are found, call `noop` with a short reason.

## Requirements

- Do not update closed pull requests.
- Do not update non-mergeable pull requests.
- Make one `update_pull_request` API call per mergeable PR.
- Keep updates deterministic and concise.

{{#import shared/noop-reminder.md}}
