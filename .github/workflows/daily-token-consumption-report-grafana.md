---
emoji: "📊"
description: Daily report of AI Credits (AIC) consumption across all agentic workflows using OTel telemetry stored in Grafana Tempo
on:
  schedule: daily on weekdays
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-token-consumption-report-grafana
engine: claude
strict: true
tools:
  bash: true
safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    title-prefix: "[token-consumption] "
    labels: [automation, observability, telemetry]
    close-older-issues: true
    expires: 1d
    max: 1
timeout-minutes: 30
imports:
  - shared/mcp/grafana.md
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[token-consumption] "
      expires: 1d
  - shared/otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily AIC Consumption Report (Grafana Tempo)

You are an observability analyst. Generate a daily AI Credits (AIC) consumption report across all agentic workflows in this repository using OpenTelemetry telemetry stored in Grafana Tempo.

## Context

- Repository: `${{ github.repository }}`
- Run ID: `${{ github.run_id }}`
- Time Window: last 24 hours

## Mission

1. Query Grafana Tempo telemetry for the last 24 hours.
2. Aggregate AIC usage by workflow.
3. Identify top AIC consumers and anomalous usage.
4. Publish a concise daily GitHub issue report.

## Data Collection

### Step 1: Discover Grafana Context

1. Call `list_datasources` to enumerate available datasources.
2. Select the Tempo datasource (type `tempo`). If multiple Tempo datasources exist, prefer the one whose name or UID suggests the production environment.
3. Note the selected datasource UID for use in subsequent queries.
4. If no Tempo datasource is found, call `noop` with an explanation and stop.

### Step 2: Discover Available Attributes

1. Call `tempo_get-attribute-names` to discover which resource and span attributes are queryable.
2. Look for AIC-related attribute names among:
   - `gh-aw.aic`
   - `gh_aw.aic`
   - `agent_usage.aic`
   - `aic`
3. Look for workflow identity attributes among:
   - `gh-aw.workflow.name`
   - `github.workflow`
   - `github.workflow_ref`
   - `gh_aw.workflow`
4. Note which attributes are actually present to guide query construction.

### Step 3: Fetch Telemetry Spans

Use `tempo_traceql-search` with the Tempo datasource UID.

**Primary query** — fetch spans with workflow attribution from this repository:

```
{ resource.github.repository = "${{ github.repository }}" }
```

Use a 24-hour time range window (current time minus 24 hours to now).

If the primary query returns no results, try progressively broader queries:

1. `{ resource.service.name =~ ".*gh-aw.*" }`
2. `{ span.gh-aw.aic > 0 }`
3. `{ }` — unfiltered, filter client-side for spans with recognized AIC or workflow fields

If `tempo_traceql-search` is unavailable, call `noop` and report that Tempo is not accessible.

**AIC-specific query** — after confirming the attribute exists, run a targeted query:

```
{ resource.github.repository = "${{ github.repository }}" && span.gh-aw.aic > 0 }
```

Use pagination to retrieve enough results to represent the last 24 hours (at least 200 results or until the query returns no more pages).

### Step 4: Verify Representative Traces

For the top 3 AIC-bearing traces found, call `tempo_get-trace` with the trace ID to:
- Confirm the trace contains AIC spans.
- Verify span attributes are present and correctly typed.
- Note any missing or unexpected attributes.

### Step 5: Extract Workflow + AIC Fields

For each span returned, derive:

- **Workflow name** using first non-empty of:
  - `gh-aw.workflow.name`
  - `github.workflow`
  - `github.workflow_ref`
  - `gh_aw.workflow`
  - fallback: `"unknown-workflow"`
- **Run ID** using:
  - `github.run_id`
  - `gh_aw.run_id`
- **AIC value** with precedence:
  - Prefer `gh-aw.aic` → `gh_aw.aic` → `agent_usage.aic` → `aic`.
  - If none are present, use `0`.

Normalize missing values to `0`.

## Analysis Requirements

Calculate:

- `total_events_analyzed`
- `events_with_aic_data`
- `events_missing_workflow`
- `total_aic`
- `workflow_count` (unique workflows)
- `top_workflows_by_aic` (top 10)
- `avg_aic_per_event`
- `p95_aic_per_event`

For each workflow include:
- workflow name
- event count
- total AIC
- average AIC/event
- highest-AIC event (with run id if available)

## Report Output

Create exactly one issue titled:

`[token-consumption] Daily AIC Consumption Report (Grafana) - YYYY-MM-DD`

**Report Formatting**: Use h3 (###) or lower for all headers to maintain proper document hierarchy. Use progressive disclosure — keep Executive Summary, Key Metrics, and Recommendations always visible, and wrap verbose details in `<details><summary>Section Name</summary>` blocks.

Use this body structure:

### Executive Summary
- Total AIC, workflow count, and high-level trend notes.

### Key Metrics
| Metric | Value |
|---|---|
| Events analyzed | ... |
| Events with AIC data | ... |
| Total AIC | ... |
| Unique workflows | ... |
| Avg AIC/event | ... |
| P95 AIC/event | ... |

### Top 10 Workflows by AIC Consumption
| Workflow | Events | Total AIC | Avg AIC/Event |
|---|---:|---:|---:|
| ... |

<details>
<summary>Representative Traces</summary>

- Include trace IDs and links for verified traces when available.
- Note any trace continuity issues.

</details>

<details>
<summary>Data Quality and Gaps</summary>

- Events missing workflow identifiers
- Events missing AIC attributes
- Attribute availability from `tempo_get-attribute-names`
- Any assumptions or fallback fields used

</details>

### Recommendations
- 2-4 concrete actions to reduce AIC usage for the highest consumers.

### References
- Include up to three relevant links (Grafana datasource links and/or run links when available).

## Guardrails

- Be explicit when telemetry fields are absent or ambiguous.
- Never invent AIC values.
- Keep the report concise and actionable.
- Use `###` or lower headers only.
- If attribute `gh-aw.aic` is absent from `tempo_get-attribute-names`, report it explicitly as a data quality gap and fall back to span volume as an activity proxy, labeled as such.

## Completion Requirement

You must call one safe output tool before finishing:
- `create_issue` for normal reporting.
- `noop` only if Tempo is unreachable or returns no usable telemetry.

{{#runtime-import shared/noop-reminder.md}}
