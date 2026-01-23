---
name: Secret Scanning Triage
description: Triage secret scanning alerts and either open an issue (rotation/incident) or a PR (test-only cleanup)
on:
  schedule: every 6h
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
engine: copilot
tools:
  github:
    toolsets: [context, repos, secret_protection, issues, pull_requests]
  repo-memory:
    - id: campaigns
      branch-name: memory/campaigns
      file-glob: [security-alert-burndown/**]
      campaign-id: security-alert-burndown
  cache-memory:
  edit:
  bash:
safe-outputs:
  add-labels:
    allowed:
      - agentic-campaign
      - z_campaign_security-alert-burndown
  create-issue:
    title-prefix: "[secret-triage] "
    labels: [security, secret-scanning, triage, agentic-campaign, z_campaign_security-alert-burndown]
    max: 1
  create-pull-request:
    title-prefix: "[secret-removal] "
    labels: [security, secret-scanning, automated-fix, agentic-campaign, z_campaign_security-alert-burndown]
    reviewers: [copilot]
timeout-minutes: 25
---

# Secret Scanning Triage Agent

You triage **one** open Secret Scanning alert per run.

## Campaign Context

This workflow is part of the **Security Alert Burndown Campaign**, which expects to find and address **21 total security findings** across the repository:
- **17 Secret scanning alerts** (this workflow addresses these)
- **3 Code scanning alerts** (handled by code-scanning-fixer and security-fix-pr workflows)
- **1 Dependabot alert** (handled by dependabot-bundler workflow)

Your focus is on the **17 secret scanning alerts**. Process them one at a time, prioritizing real credentials that need rotation over test-only secrets.

## Guardrails

- Always operate on `owner="githubnext"` and `repo="gh-aw"`.
- Do not dismiss alerts unless explicitly instructed (this workflow does not have a dismiss safe output).
- Prefer a PR only when the secret is clearly **test-only / non-production** (fixtures, tests, sample strings) and removal is safe.
- If it looks like a real credential, open an issue with rotation steps.

## State tracking

Use cache-memory file `/tmp/gh-aw/cache-memory/secret-scanning-triage.jsonl`.

- Each line is JSON: `{ "alert_number": 123, "handled_at": "..." }`.
- Treat missing file as empty.

## Steps

### 1) List open secret scanning alerts

Use the GitHub MCP `secret_protection` toolset.

- Call `github-list_secret_scanning_alerts` (or the closest list tool in the toolset) for `owner="githubnext"` and `repo="gh-aw"`.
- Filter to `state="open"`.

If none, log and exit.

### 2) Pick the next unhandled alert

- Load handled alert numbers from cache-memory.
- Pick the first open alert that is not in the handled set.
- If all are handled, log and exit.

### 3) Fetch details + location

Use the appropriate tool (e.g. `github-get_secret_scanning_alert` and/or an “alert locations” tool if available) to collect:
- alert number
- secret type (if present)
- file path and commit SHA (if present)
- a URL to the alert

### 4) Classify

Classify into one of these buckets:

A) **Test/sample string**
- Path contains: `test`, `tests`, `fixtures`, `__tests__`, `testdata`, `examples`, `docs`, `slides`
- The string looks like a fake token (obvious placeholders) OR is used only in tests

B) **Likely real credential**
- Path is in source/runtime code (not tests/docs)
- The token format matches a real provider pattern and context suggests it is authentic

If unsure, treat as (B).

### 5A) If (A): create a PR removing/replacing the secret

- Check out the repository.
- Make the smallest change to remove the secret:
  - Replace with a placeholder like `"REDACTED"` or `"<TOKEN>"`
  - If tests require it, add a deterministic fake value and adjust test expectations
- Run the most relevant lightweight checks (e.g. `go test ./...` if Go files changed, or the repo’s standard test command if obvious).

Then emit one `create_pull_request` safe output with:
- What you changed
- Why it’s safe
- Link to the alert

### 5B) If (B): create an issue with rotation steps

Emit one `create_issue` safe output with:
- Alert link
- File path(s)
- Recommended immediate actions:
  - rotate the credential
  - invalidate the old token
  - audit recent usage
  - then remove from repo history if applicable
- Suggested follow-up: add detection/guardrails (e.g. pre-commit secret scanning)

### 6) Record handling

Append a JSON line to `/tmp/gh-aw/cache-memory/secret-scanning-triage.jsonl` for the alert you handled.
