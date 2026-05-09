---
description: Investigates [aw] failures from the last 6 hours, correlates with open agentic-workflows issues, closes fixed issues, and opens focused fix sub-issues when needed
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
  bash: ["*"]
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[aw-failures] "
    labels: [agentic-workflows, automation, cookie]
    max: 2
    group: true
  update-issue:
    target: "*"
    max: 10
  link-sub-issue:
    max: 10
  noop:
timeout-minutes: 60
imports:
  - uses: shared/meta-analysis-base.md
    with:
      toolsets: [default, actions]
  - shared/reporting.md

  - shared/observability-otlp.md
features:
  inline-agents: true
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
4. Close fixed/stale issues first, then create only the minimum necessary linked fix sub-issues.

## Required Investigation Steps

### 1) Fetch and review existing issue context

Use the `issue-context-fetcher` agent to retrieve open `agentic-workflows` issues grouped into clusters, gaps, and potential duplicates. Use the returned JSON when correlating failures.

### 2) Collect workflow runs and isolate failures (last 6h)

Use the `failure-dataset-builder` agent to fetch logs for the last 6 hours and return clustered failure rows with representative + comparator run IDs.

### 3) Deep-dive each failure cluster with `audit`

Use the `cluster-evidence-extractor` agent, passing the clusters from step 2, to retrieve per-cluster evidence (dominant error, tool-failure pattern, anomalies, failure class).

### 4) Compare behavior with `audit-diff`

Use `agentic-workflows` MCP `audit-diff` to compare:
- failed run vs nearest successful run of the same workflow, or
- failed run vs prior failed run to detect drift

Identify regressions and deltas (metrics/tooling/firewall/MCP behavior) that support fix recommendations.

### 5) Close fixed issues first, then add focused sub-issues

First, identify currently open `agentic-workflows` issues that are now fixed, stale, or no longer actionable based on fresh evidence, and close them using `update-issue`.

Then, if new uncovered work remains, add **sub-issues** for concrete fixes to the **most recent open parent report issue** instead of creating a new parent by default.

Only create a new parent report issue (temporary ID format `aw_` + 3-8 alphanumeric characters) when **P0 failures have no existing tracking coverage**.

Each new sub-issue must include:
- clear problem statement
- affected workflows and run IDs
- probable root cause
- specific proposed remediation
- success criteria / verification

## Output Requirements

**Report Formatting**: Use `###` or lower for all headers in the issue body. Wrap evidence/log excerpts and verbose tables in `<details><summary>Section Name</summary>` tags.

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
- If failures exist but are already fully tracked, prefer closing stale/fixed issues and avoid creating new issues.
- Only create a new parent report issue when P0 failures have no existing tracking coverage.
- Prefer closing stale/fixed issues over creating new issues when issue volume is high.
- Always be explicit about confidence and unknowns.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```

## agent: `issue-context-fetcher`
---
description: Fetches open agentic-workflows issues and groups them into clusters, gaps, and duplicate candidates
model: small
---
Find open issues labeled `agentic-workflows` for `${{ github.repository }}`.
Group findings into existing tracked clusters, tracking gaps, and potential duplicates.

Return only JSON:
```json
{
  "clusters": [{"name":"", "issue_numbers":[]}],
  "gaps": [{"failure_signature":"", "reason":""}],
  "potential_duplicates": [{"issue_numbers":[], "reason":""}]
}
```

## agent: `failure-dataset-builder`
---
description: Fetches the last 6h workflow logs and builds clustered failure rows with representative and comparator run IDs
model: small
---
Use `agentic-workflows` MCP `logs` for the last 6 hours (for example `start_date: "-6h"`), including enough runs to cover the window.
Cluster failures by signature and include representative and comparator run IDs.

Return only JSON:
```json
{
  "failure_rows": [{"cluster_id":"", "workflow":"", "engine":"", "failure_signature":"", "representative_failed_run_id":"", "comparator_success_run_id":"", "run_ids":[]}]
}
```

## agent: `cluster-evidence-extractor`
---
description: Extracts per-cluster audit evidence including dominant errors, tool patterns, anomalies, and failure class
model: small
---
Given failure clusters from step 2, call `agentic-workflows` MCP `audit` for each cluster's representative failed run and a successful comparator when available.
Extract dominant error, tool-failure pattern, anomalies, and failure class.

Return only JSON:
```json
{
  "cluster_evidence": [{"cluster_id":"", "dominant_error":"", "tool_failure_pattern":"", "anomalies":[],"failure_class":"","evidence_run_ids":[]}]
}
```
