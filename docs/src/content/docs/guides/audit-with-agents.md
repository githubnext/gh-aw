---
title: Consuming Audit Reports with Agents
description: How to feed structured audit output into agentic workflows for automated triage, trend analysis, and remediation.
---

When running locally, all three audit commands accept `--json` to write structured output to stdout. Pipe through `jq` to extract the fields a model needs.

| Command | Use case |
|---------|----------|
| `gh aw audit <run-id> --json` | Single run — `key_findings`, `recommendations`, `metrics` |
| `gh aw logs [workflow] --last 10 --json` | Trend analysis — `per_run_breakdown`, `domain_inventory` |
| `gh aw audit diff <id1> <id2> --json` | Before/after — `run_metrics_diff`, `firewall_diff` |

Inside GitHub Actions workflows, agents access these commands through the `agentic-workflows` MCP tool rather than calling the CLI directly.

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

Use the `agentic-workflows` MCP tool `audit` with run ID ${{ github.event.workflow_run.id }}, identify the pull request that triggered it, and post a comment summarizing key findings and blocked domains. Highlight issues with severity `high` or `critical`. If there are no findings, post a brief "no issues found" comment.
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

Use the `agentic-workflows` MCP tool `audit diff` with base run ID ${{ inputs.base_run_id }} and current run ID ${{ inputs.current_run_id }}. Check for new blocked domains, increased MCP error rates, cost increase > 20%, or token usage increase > 50%. If regressions are found, open a GitHub issue with a table from `run_metrics_diff`, affected domains from `firewall_diff`, and affected MCP tools from `mcp_tools_diff`.
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

Use the `agentic-workflows` MCP tool `audit` with run ID ${{ github.event.workflow_run.id }}. Filter `key_findings` for severity `high` or `critical`. For each finding without a matching open issue, create one with the finding title, description, impact, and recommendations, labelled `audit-finding`. If no critical findings, call the `noop` safe output tool.
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

1. Use the `agentic-workflows` MCP tool `logs` with parameters `workflow: my-workflow, last: 10` and read `/tmp/gh-aw/cache-memory/audit-trends.json` as the previous baseline.
2. Detect: cost spikes (`cost_spike: true` in `per_run_breakdown`), new denied domains in `domain_inventory`, MCP servers with `error_rate > 0.10` or `unreliable: true`, and week-over-week changes in `error_trend.runs_with_errors`.
3. Create a GitHub discussion "Audit Digest — [YYYY-MM-DD]" with an executive summary, anomalies table, and MCP health table.
4. Update `/tmp/gh-aw/cache-memory/audit-trends.json` with rolling averages (cost, tokens, error count, deny rate), keeping only the last 30 days.
```

## Tips

Top-level fields (`key_findings`, `recommendations`, `metrics`, `firewall_analysis`, `mcp_tool_usage`) are stable; nested sub-fields may be extended but are not removed without deprecation. Add `--parse` to populate `behavior_fingerprint` and `agentic_assessments`. Cross-run JSON can be large — extract only the slices your model needs.
