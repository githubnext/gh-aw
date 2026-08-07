---
private: true
emoji: "🧭"
description: Daily analysis of recent CI test failures and regression patterns using Kiro
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
sandbox:
  agent:
    sudo: false
tracker-id: daily-regression-audit-kiro
engine:
  id: kiro
model: kiro/claude-sonnet-4-5
strict: true
network:
  allowed:
    - defaults
    - github
tools:
  github:
    mode: local
    toolsets: [repos, issues, actions]
  bash:
    - cat
    - grep
    - wc
safe-outputs:
  create-issue:
    expires: 2d
    title-prefix: "[regression] "
    labels: [automation, testing, reliability]
    max: 1
    close-older-issues: true
    close-older-key: daily-regression-audit-kiro
  missing-tool:
timeout-minutes: 25
imports:
  - shared/kiro.md
  - shared/otlp.md
---

# Daily Regression Audit — Kiro

Analyze recent CI workflow runs to surface patterns in test failures and regressions.

## Step 1 — Fetch recent workflow runs

Use the GitHub MCP `list_workflow_runs` tool on `${{ github.repository }}` with:
- `workflow_id: ci.yml` (or the main CI workflow)
- `per_page: 20`
- `status: completed`

Record the last 20 run conclusions (success / failure / cancelled).

## Step 2 — Identify failure rate

Calculate:
- Total runs in the set
- Failure count and percentage
- Consecutive failure streak (if any)

If failure rate is below 10% and no streak exists, skip to Step 4 with a "healthy" conclusion.

## Step 3 — Inspect failed runs

For the most recent 3 failed runs, use `list_workflow_jobs` to get the jobs and then
`get_job_logs` to retrieve failing job output (tail 200 lines). Look for:
- Repeated error messages across runs (regression patterns)
- New failures that appear only in recent runs (potential regressions introduced recently)
- Flaky tests vs. consistent failures

Summarize the top 2–3 failure patterns found.

## Step 4 — Check for open regression issues

Use `list_issues` on `${{ github.repository }}` with `labels: ["bug","regression"]` and
`state: open`, `perPage: 5`. Record issue numbers and titles.

## Step 5 — Report

Use the `create_issue` safe-output tool to post the daily audit:

- **Title**: `[regression] Daily CI Regression Audit — ${{ github.run_id }}`
- **Body**: Summarize
  - CI failure rate (last 20 runs)
  - Top failure patterns or "No regressions detected"
  - Open regression/bug issues
  - Recommended action (investigate / monitor / no action)

If CI is fully healthy (failure rate < 10%, no streak, no new patterns), call `noop` with
`"CI health is good — no regressions detected."`.
