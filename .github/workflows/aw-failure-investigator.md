---
description: Investigates [aw] failures from the last 6 hours, correlates with open agentic-workflows issues, and opens a parent report with fix sub-issues
on:
  schedule:
    - cron: "every 6h"
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: aw-failure-investigator
engine: claude
tools:
  mount-as-clis: true
  agentic-workflows:
  github:
    toolsets: [default, actions]
  bash: ["*"]
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[aw-failures] "
    labels: [agentic-workflows, automation, cookie]
    max: 8
    group: true
  link-sub-issue:
    max: 20
  noop:
timeout-minutes: 60
imports:
  - shared/reporting.md
features:
  mcp-cli: true
---

# [aw] Failure Investigator (6h)

Investigate agentic workflow failures from the last 6 hours and produce actionable issue tracking with sub-issues.

## Scope

- **Repository**: `${{ github.repository }}`
- **Lookback window**: last 6 hours
- **Issue query to inspect first**: <https://github.com/github/gh-aw/issues?q=is%3Aissue%20state%3Aopen%20label%3Aagentic-workflows>

## Mission

1. Find recent failures from agentic workflows in the last 6 hours.
2. Correlate findings with currently open `agentic-workflows` issues.
3. Perform large-scale failure analysis using logs + audit + audit-diff.
4. Create one parent report issue and linked sub-issues proposing concrete fixes.

## Required Investigation Steps

### 1) Fetch and review existing issue context

Find open issues with `agentic-workflows` label using GitHub issues search (equivalent to the URL query above). Focus on issues created/updated in the lookback window and unresolved recurring themes.

Capture:
- Existing failure clusters already tracked
- Gaps where recurring failures are not yet tracked
- Potential duplicates to avoid

### 2) Collect workflow runs and isolate failures (last 6h)

Use `agentic-workflows` MCP `logs` with a 6-hour window (for example `start_date: "-6h"`) and enough count to cover all recent runs.

Build a failure dataset including:
- run id, workflow, engine, status/conclusion
- timestamps and durations
- repeated failure signatures
- affected tools / MCP / firewall / auth / timeout dimensions

### 3) Deep-dive each failure cluster with `audit`

For each meaningful cluster (not every single run if many are equivalent), call `agentic-workflows` MCP `audit` on representative failed runs and at least one successful comparator run when available.

Extract evidence:
- root-cause signals
- dominant error messages
- tool failure patterns
- token/cost/runtime anomalies
- infra vs workflow-definition vs prompt/tooling failure classification

### 4) Compare behavior with `audit-diff`

Use `agentic-workflows` MCP `audit-diff` to compare:
- failed run vs nearest successful run of the same workflow, or
- failed run vs prior failed run to detect drift

Identify regressions and deltas (metrics/tooling/firewall/MCP behavior) that support fix recommendations.

### 5) Create parent report issue + sub-issues

Create a **single parent report issue** with a temporary ID (format `aw_` + 3-8 alphanumeric characters) summarizing:
- observed failure clusters in last 6h
- links to analyzed run IDs
- evidence from logs/audit/audit-diff
- mapping to existing open issues (duplicate / related / new)
- prioritized fix plan

Then create **sub-issues** (linked to the parent) for concrete fixes. Each sub-issue must include:
- clear problem statement
- affected workflows and run IDs
- probable root cause
- specific proposed remediation
- success criteria / verification

## Output Requirements

### Parent report issue structure

Include these sections:
1. Executive summary
2. Failure clusters (table)
3. Evidence (logs/audit/audit-diff)
4. Existing issue correlation
5. Proposed fix roadmap (P0/P1/P2)
6. Sub-issues created

### Sub-issue quality bar

- Prefer a few high-quality, actionable sub-issues over many weak ones.
- Avoid duplicates of already-open issues unless new evidence materially changes scope.
- Reference the parent issue and the concrete run IDs analyzed.

## Decision Rules

- If there are **no failures** in the last 6h, or no actionable delta vs existing issues, call `noop` with a concise reason.
- If failures exist but are already fully tracked, update by creating a minimal parent report that links to existing issues and only create new sub-issues for uncovered gaps.
- Always be explicit about confidence and unknowns.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
