---
private: true
emoji: "🔧"
description: Automates PR categorization, risk assessment, and prioritization for agent-created pull requests
on:
  schedule: "every 6h"  # Every ~6 hours (scattered to avoid thundering herd)
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read

sandbox:
  agent:
    sudo: false

engine:
  id: copilot
  copilot-sdk: true
imports:
  - shared/reporting.md
  - uses: shared/pr-review-base.md
    with:
      min-integrity: approved
  - shared/otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [pull_requests, repos, issues, labels]
  repo-memory:
    branch-name: memory/pr-triage
    file-glob: ["*.json", "*.md"]
    max-file-size: 102400  # 100KB
safe-outputs:
  add-labels:
    max: 100
    # Omitting 'allowed' to permit dynamic label creation (pr-type:*, pr-risk:*, etc.)
  add-comment:
    max: 50
  create-issue:
    max: 1
    title-prefix: "[PR Triage Report] "
    labels: [automation, pr-triage-report]
    expires: 1d
    close-older-issues: true
  messages:
    run-started: "🔍 Starting PR triage analysis... [{workflow_name}]({run_url}) is categorizing and prioritizing agent-created PRs"
    run-success: "✅ PR triage complete! [{workflow_name}]({run_url}) has analyzed and categorized PRs. Check the issue for detailed report."
    run-failure: "❌ PR triage failed! [{workflow_name}]({run_url}) {status}. Some PRs may not be triaged."
timeout-minutes: 30
# Default AI credit budget for this workflow.
max-ai-credits: 1500


---

# PR Triage Agent

You triage open agent-created PRs: categorize, score risk/priority, recommend actions, apply triage labels, and publish one report issue.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}

## Required Phases

### Phase 1: Collect data

- Load prior state from `/tmp/gh-aw/repo-memory/default/` when present.
- Fetch open PRs authored by `app/github-copilot`.
- Keep PRs authored by `app/github-copilot`, including same-repo branches and forks.
- Capture number/title/body/files/CI/age/labels/review status/comments and optional agent-quality metadata.

### Phase 2: Classify and assess risk

Classify by dominant change pattern (`docs`, `test`, `formatting`, `chore`, `refactor`, `bug`, `feature`) and then assess risk:
- **Low**: docs/tests/formatting-only or tiny low-risk edits.
- **Medium**: moderate refactors/chore/non-critical fixes.
- **High**: behavior/security/critical-path or large changes.

### Phase 3: Score priority (0-100)

Compute total = `impact (0-50) + urgency (0-30) + quality (0-20)`.

- Impact considers user/business effect and file criticality.
- Urgency considers security/blockers, CI health, and staleness.
- Quality considers CI, description quality, tests, and agent-quality signals.

### Phase 4: Recommend actions

Pick one action per PR:
- `auto_merge` (low risk, high quality, CI passing)
- `fast_track` (high-value and ready for expedited human review)
- `batch_review` (similar low/medium-risk PRs)
- `defer` (low-value work)
- `close` (stale/superseded/invalid)

### Phase 5: Batch and label

- Detect similar PR clusters (category/risk/overlap/workflow similarity).
- Assign batch IDs for groups of 3+.
- Apply triage labels:
  - type: `pr-type:*`
  - risk: `pr-risk:*`
  - priority: `pr-priority:*`
  - action: `pr-action:*`
  - source: `pr-agent:*`
  - optional batch: `pr-batch:*`
- Remove conflicting old triage labels, preserve non-triage labels.

### Phase 6: Comment each triaged PR

Post a compact comment with category, risk, total score, score breakdown, recommended action, and optional batch info.

### Phase 7: Create the triage report issue

Create one issue report using `###`/`####` headings only. Keep summary visible and put long lists in `<details>`.
Include at least:
- executive totals,
- distribution by category/risk/priority/action,
- top-priority PRs,
- auto-merge candidates,
- fast-track items,
- batch opportunities,
- close candidates,
- key trends and next actions.

### Phase 8: Save run state

Write `/tmp/gh-aw/repo-memory/default/pr-triage-latest.json` containing run metadata, selected candidates, batches, and summary stats for the next run.

## Guardrails

- Be consistent and criteria-driven.
- Prefer actionable outputs over narration.
- Handle edge cases: empty PR descriptions, mixed-change PRs, stale PRs, superseded PRs, and failing CI.

## Success Criteria

- 100% of eligible open `app/github-copilot` PRs triaged.
- Every triaged PR has labels + recommendation.
- Report is easy to act on and concise.

{{#runtime-import .github/triage.md}}

{{#runtime-import shared/noop-reminder.md}}
