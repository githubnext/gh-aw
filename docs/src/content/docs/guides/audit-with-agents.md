---
title: Consuming Audit Reports with Agents
description: How to feed structured audit output into agentic workflows for automated triage, trend analysis, and remediation.
---

The audit commands produce structured JSON that agents can consume programmatically for automated triage, cost monitoring, and incident response. This guide shows how to connect audit data to workflow agents.

## Getting structured audit data

All three audit commands support `--json`, which writes structured output to stdout:

```bash
# Single run audit
gh aw audit <run-id> --json

# Cross-run analysis
gh aw logs [workflow] --last 10 --json

# Before/after comparison
gh aw audit diff <run-id-1> <run-id-2> --json
```

### Key fields for agent consumption

| Field | Description |
|-------|-------------|
| `key_findings` | Categorized issues with severity and impact |
| `recommendations` | Prioritized actions with example fixes |
| `firewall_analysis` | Network request stats per domain |
| `mcp_tool_usage` | Per-tool invocation counts and error rates |
| `metrics` | Token usage, estimated cost, and run duration |
| `errors` / `warnings` | Structured error details with file and line |

Use `jq` to extract only the fields an agent needs before passing to a model:

```bash
# Key findings and recommendations only
gh aw audit <run-id> --json | jq '{findings: .key_findings, recommendations: .recommendations}'

# Domains that were blocked
gh aw audit <run-id> --json | jq '.firewall_analysis.domains[] | select(.blocked > 0)'

# MCP tools with errors
gh aw audit <run-id> --json | jq '.mcp_tool_usage.summary[] | select(.error_count > 0)'
```

For cross-run reports, extract the fields relevant to trend analysis:

```bash
# Per-run cost and token data
gh aw logs my-workflow --last 10 --json | jq '.per_run_breakdown[] | {run_id, cost, tokens, turns}'

# Domain inventory showing policy status across runs
gh aw logs my-workflow --last 10 --json | jq '.domain_inventory[] | {domain, overall_status, seen_in_runs}'
```

## Feeding audit data into a workflow agent

### Post findings as a review comment

This workflow runs after each completed agent run and posts audit findings as a pull request comment:

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

Run ID: ${{ github.event.workflow_run.id }}

1. Fetch the audit report for run ${{ github.event.workflow_run.id }} using the `audit` tool
2. Identify the pull request that triggered this workflow run
3. Post a comment summarizing the key findings, any blocked domains, and MCP tool errors
4. Highlight critical issues (severity: high or error) that need immediate attention
5. If there are no findings, post a brief "no issues found" comment
```

### Detect regressions with diff

This workflow compares a baseline run against a current run and opens an issue if regressions are found:

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

Compare run ${{ inputs.base_run_id }} (baseline) against ${{ inputs.current_run_id }} (current).

1. Run `gh aw audit diff ${{ inputs.base_run_id }} ${{ inputs.current_run_id }} --json` using the shell tool
2. Check for: new blocked domains, increased MCP error rates, cost increase > 20%, or token usage increase > 50%
3. If regressions are found, open a GitHub issue titled "Regression detected in [workflow name]" with:
   - A table of changes from `run_metrics_diff`
   - List of new or changed domains from `firewall_diff`
   - Affected MCP tools from `mcp_tools_diff`
4. If no regressions are found, output a summary confirming stable behavior
```

### Auto-file issues from audit findings

This workflow runs `gh aw audit` after each agent run and creates GitHub issues for high-severity findings:

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

Run ID: ${{ github.event.workflow_run.id }}

1. Fetch the audit report for run ${{ github.event.workflow_run.id }} using the `audit` tool
2. Filter `key_findings` for entries with severity `high` or `critical`
3. For each critical finding, check if a GitHub issue with the same title already exists
4. If no duplicate exists, create an issue with:
   - Title: the finding title
   - Body: description, impact, and recommendations from the audit report
   - Label: `audit-finding`
5. If no critical findings, call the `noop` safe output tool
```

## Building an audit monitoring agent

This full example monitors a workflow over time, detecting cost spikes, new blocked domains, and error rate increases, then posts a weekly digest:

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

Workflow to monitor: my-workflow

## Step 1: Collect data

Run `gh aw logs my-workflow --last 10 --json` using the shell tool and capture the output.

## Step 2: Load previous trends

Read `/tmp/gh-aw/cache-memory/audit-trends.json` if it exists (previous week's baseline).

## Step 3: Analyze trends

Compare current data against the baseline to detect:

- **Cost spikes**: runs where `cost > 2× average` (indicated by `cost_spike: true` in `per_run_breakdown`)
- **New blocked domains**: domains in `domain_inventory` with `overall_status: denied` not present in the baseline
- **MCP reliability**: servers in `mcp_health` with `error_rate > 0.10` or `unreliable: true`
- **Error trend**: check if `error_trend.runs_with_errors` is increasing week-over-week

## Step 4: Post discussion

Create a GitHub discussion titled "Audit Digest — [date]" with:
- Executive summary: runs analyzed, total cost, avg tokens, overall deny rate
- Anomalies table: any spikes or new blocked domains
- MCP health table: servers with elevated error rates
- Trend direction (improving / stable / degrading) based on comparison

## Step 5: Update cache

Write updated aggregate metrics to `/tmp/gh-aw/cache-memory/audit-trends.json`:
- Use filesystem-safe timestamps (YYYY-MM-DD, not ISO 8601 with colons)
- Store rolling averages for cost, tokens, error count, and deny rate
- Keep only the last 30 days of data to limit cache size
```

> [!TIP]
> Store aggregate metrics (rolling averages, domain counts) in `cache-memory` rather than full audit JSON. Full cross-run reports can be large; caching only the summary fields keeps well within GitHub Actions cache limits.

## Tips

**JSON schema stability**: The top-level fields (`key_findings`, `recommendations`, `metrics`, `firewall_analysis`, `mcp_tool_usage`) are stable. Nested sub-fields may be extended in minor releases but are not removed without deprecation. Pin your `jq` filters to the fields you rely on and treat unknown fields as optional.

**Combining with `--parse`**: Add `--parse` to run log parsers before generating JSON output. This populates `behavior_fingerprint` and `agentic_assessments`, which give agents richer context for behavioral analysis and pattern detection.

**Before/after optimization**: Use `gh aw audit diff` in optimization workflows to verify that prompt or configuration changes reduced cost and domain access without introducing new errors. The `run_metrics_diff.cost_change` and `run_metrics_diff.token_usage_change` fields give direct before/after comparisons.

**Filtering for context windows**: Cross-run JSON from `gh aw logs --json` can be large. Extract only the fields your agent needs — for example, `per_run_breakdown` for cost tracking or `domain_inventory` for firewall policy analysis — before passing to a model with a limited context window.
