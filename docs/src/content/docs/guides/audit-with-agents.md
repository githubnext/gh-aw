---
title: Consuming Audit Reports with Agents
description: How to feed structured audit output into agentic workflows for automated triage, trend analysis, and remediation.
---

All three audit commands support `--json`, which writes structured output to stdout.

```bash
gh aw audit <run-id> --json                         # single run
gh aw logs [workflow] --last 10 --json              # cross-run analysis
gh aw audit diff <run-id-1> <run-id-2> --json       # before/after comparison
```

Key fields for agent consumption: `key_findings`, `recommendations`, `firewall_analysis`, `mcp_tool_usage`, `metrics`, `errors`. Use `jq` to extract only what the model needs:

```bash
gh aw audit <run-id> --json | jq '{findings: .key_findings, recommendations: .recommendations}'
gh aw audit <run-id> --json | jq '.firewall_analysis.domains[] | select(.blocked > 0)'
gh aw logs my-workflow --last 10 --json | jq '.per_run_breakdown[] | {run_id, cost, tokens, turns}'
```

## Posting findings as a PR comment

```aw wrap
---
description: Post audit findings as a PR comment after each agent run
on:
  workflow_run:
    workflows: ['my-workflow']
    types: [completed]
engine: copilot
tools:
  github:
    toolsets: [pull_requests]
  agentic-workflows:
permissions:
  contents: read
  actions: read
  pull-requests: write
---

# Summarize Audit Findings

Fetch the audit report for run ${{ github.event.workflow_run.id }}, identify the pull request that triggered it, and post a comment summarizing key findings and blocked domains. Highlight issues with severity `high` or `critical`. If there are no findings, post a brief "no issues found" comment.
```

## Detecting regressions with diff

```aw wrap
---
description: Detect regressions between two workflow runs
on:
  workflow_dispatch:
    inputs:
      base_run_id:
        description: 'Baseline run ID'
        required: true
      current_run_id:
        description: 'Current run ID to compare'
        required: true
engine: copilot
tools:
  github:
    toolsets: [issues]
  agentic-workflows:
permissions:
  contents: read
  actions: read
  issues: write
---

# Regression Detection

Run `gh aw audit diff ${{ inputs.base_run_id }} ${{ inputs.current_run_id }} --json`. Check for new blocked domains, increased MCP error rates, cost increase > 20%, or token usage increase > 50%. If regressions are found, open a GitHub issue with a table from `run_metrics_diff`, affected domains from `firewall_diff`, and affected MCP tools from `mcp_tools_diff`.
```

## Filing issues from audit findings

```aw wrap
---
description: File GitHub issues for high-severity audit findings
on:
  workflow_run:
    workflows: ['my-workflow']
    types: [completed]
engine: copilot
tools:
  github:
    toolsets: [issues]
  agentic-workflows:
permissions:
  contents: read
  actions: read
  issues: write
---

# Auto-File Issues for Critical Findings

Fetch the audit report for run ${{ github.event.workflow_run.id }}. Filter `key_findings` for severity `high` or `critical`. For each finding without a matching open issue, create one with the finding title, description, impact, and recommendations, labelled `audit-finding`. If no critical findings, call the `noop` safe output tool.
```

## Weekly audit monitoring agent

```aw wrap
---
description: Weekly audit digest with trend analysis
on:
  schedule: weekly
engine: copilot
tools:
  github:
    toolsets: [discussions]
  agentic-workflows:
  cache-memory:
    key: audit-monitoring-trends
permissions:
  contents: read
  actions: read
  discussions: write
---

# Weekly Audit Monitoring Digest

1. Run `gh aw logs my-workflow --last 10 --json` and read `/tmp/gh-aw/cache-memory/audit-trends.json` as the previous baseline.
2. Detect: cost spikes (`cost_spike: true` in `per_run_breakdown`), new denied domains in `domain_inventory`, MCP servers with `error_rate > 0.10` or `unreliable: true`, and week-over-week changes in `error_trend.runs_with_errors`.
3. Create a GitHub discussion "Audit Digest — [YYYY-MM-DD]" with an executive summary, anomalies table, and MCP health table.
4. Update `/tmp/gh-aw/cache-memory/audit-trends.json` with rolling averages (cost, tokens, error count, deny rate), keeping only the last 30 days.
```

## Tips

The top-level fields (`key_findings`, `recommendations`, `metrics`, `firewall_analysis`, `mcp_tool_usage`) are stable across releases; nested sub-fields may be extended but are not removed without deprecation.

Add `--parse` to populate `behavior_fingerprint` and `agentic_assessments` for richer behavioral context.

Cross-run JSON from `gh aw logs --json` can be large — extract only the fields needed (e.g. `per_run_breakdown`, `domain_inventory`) before passing to a model.
