---
emoji: "📊"
description: Daily report of token consumption across all agentic workflows using OTel telemetry stored in Sentry
on:
  schedule: daily on weekdays
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-token-consumption-report
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
  - shared/mcp/sentry.md
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[token-consumption] "
      expires: 1d

  - shared/otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily AIC Consumption Report (Sentry OTel)

You are an observability analyst. Generate a daily AI Credits (AIC) consumption report across all agentic workflows in this repository using OpenTelemetry telemetry in Sentry.

## Context

- Repository: `${{ github.repository }}`
- Run ID: `${{ github.run_id }}`
- Time Window: last 24 hours

## Mission

1. Query Sentry telemetry for the last 24 hours.
2. Aggregate AIC usage by workflow.
3. Identify top AIC consumers and anomalous usage.
4. Publish a concise daily GitHub issue report.

## Data Collection

### Step 1: Discover Sentry Context

1. Call `find_organizations` and select the org for this repository.
2. Call `find_projects` and select the project that corresponds to `${{ github.repository }}`.

### Step 2: Fetch Telemetry Events

First, attempt to call `search_events` using:
- `dataset: spans`
- query constrained to the selected project
- time range: last 24 hours
- include enough results to cover the day (use pagination as needed)

If `search_events` is **not available** (the tool is absent from the available tool list because no embedded LLM provider is configured), fall back to `list_events` with direct Sentry query syntax:
- `organizationSlug`: the org discovered in Step 1
- `dataset: spans`
- `query`: a filter for AI/LLM spans with AIC/cost data; start with `span.op:ai*` and also try `span.op:gen_ai*` if the first returns no results; if neither matches, try an empty query and filter client-side for records with AIC fields
- `fields`: include AIC/cost fields such as `gh-aw.aic`, `gh_aw.aic`, `aic`, `agent_usage.aic`, plus workflow metadata (`github.workflow`, `github.run_id`, `span.op`, `span.description`, `timestamp`); omit fields that return errors and retry with remaining fields
- `sort`: `-timestamp`
- Use pagination to cover the last 24 hours

If `dataset: spans` returns no usable records with either tool, retry with `dataset: transactions`.

Treat "no usable records" as either:
- zero events returned after pagination, or
- events returned but none contain any recognized AIC fields.

### Step 3: Extract Workflow + AIC Fields

For each event/span, derive:

- **Workflow name** using first non-empty of likely fields:
  - `github.workflow`
  - `github.workflow_ref`
  - `workflow.name`
  - `gh_aw.workflow`
  - fallback: `"unknown-workflow"`
- **Run ID** using:
  - `github.run_id`
  - `gh_aw.run_id`
- **AIC value** with precedence:
  - Prefer `gh-aw.aic` → `gh_aw.aic` → `agent_usage.aic` → `aic`.
  - If none are present, use `0`.
- Recognized AIC fields:
  - `gh-aw.aic`
  - `gh_aw.aic`
  - `agent_usage.aic`
  - `aic`

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

`[token-consumption] Daily AIC Consumption Report - YYYY-MM-DD`

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
<summary>Data Quality and Gaps</summary>

- Events missing workflow identifiers
- Events missing AIC attributes
- Any assumptions or fallback fields used

</details>

### Recommendations
- 2-4 concrete actions to reduce AIC usage for the highest consumers.

### References
- Include up to three relevant links (Sentry query links and/or run links when available).

## Guardrails

- Be explicit when telemetry fields are absent or ambiguous.
- Never invent AIC values.
- Keep the report concise and actionable.
- Use `###` or lower headers only.

## Completion Requirement

You must call one safe output tool before finishing:
- `create_issue` for normal reporting.
- `noop` only if no valid telemetry could be retrieved.

{{#runtime-import shared/noop-reminder.md}}
