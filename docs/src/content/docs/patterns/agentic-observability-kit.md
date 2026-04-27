---
title: Agentic Observability Kit
description: Use the built-in Agentic Observability Kit to review agentic workflow behavior, detect regressions, and identify evidence-based repository portfolio cleanup opportunities.
---

> [!WARNING]
> **Experimental:** The Agentic Observability Kit is still experimental! Things may break, change, or be removed without deprecation at any time.

The Agentic Observability Kit reviews recent agentic workflow runs in a repository and produces one operator-facing report. It reads run history, episode rollups, and selective audit details, posts a discussion with the results and opens one escalation issue only when repeated patterns warrant owner action.

The kit now also includes an evidence-based repository portfolio review. In the same report, maintainers can call out workflows that look stale, overlapping, weakly justified for their recent cost, or consistently overbuilt for the task domain they serve.

This pattern is useful when a repository has enough agentic activity that per-run inspection is too noisy, but maintainers still need practical answers to questions such as which workflows are drifting, which runs are expensive for their domain, which orchestrated chains are accumulating risk, and which workflows may no longer justify their current form.

## Scope

The built-in workflow is repository-scoped. The report combines two layers:

- operational observability for recent runs, episodes, regressions, and control failures
- an evidence-based portfolio appendix for overlap, stale workflows, and weakly justified agentic workflows

The same pattern extends to organization and enterprise scope via central repository aggregation — see [Deployment by scope](#deployment-by-scope) below. Organization-wide and enterprise-wide deployments require additional cross-repository authentication, central orchestration, and portfolio aggregation logic beyond the single built-in workflow.

## Deployment by scope

The practical setup differs by scope.

### Single repository: install the built-in workflow

For one repository, use the built-in workflow directly and keep the report local to that repository.

```aw wrap
---
on:
  schedule: weekly on monday around 08:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read
engine: copilot
tools:
  agentic-workflows:
  github:
    toolsets: [default, discussions]
safe-outputs:
  create-issue:
    title-prefix: "[observability escalation] "
    max: 1
---

# Agentic Observability Kit

Review recent agentic workflow runs in this repository and publish one discussion-oriented report.
```

This is the right default when maintainers want repository-local visibility and repository-local ownership, including an evidence-based review of whether current workflows still look justified for the repository.

### One organization: aggregate from a central repository

For an organization, prefer one central repository that discovers target repositories, pulls per-repository observability data, and publishes an organization-level rollup.

```aw wrap
---
on:
  schedule: weekly on monday around 09:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  discussions: read
engine: copilot
tools:
  github:
    github-token: ${{ secrets.GH_AW_READ_ORG_TOKEN }}
    toolsets: [repos]
  bash:
    - "gh aw logs *"
safe-outputs:
  create-discussion:
    max: 1
---

# Organization Observability Rollup

Discover target repositories, collect per-repository `gh aw logs --json --repo owner/repo` output, and generate one organization-level summary.
```

This is the recommended org-wide model because it centralizes authentication, repository allowlists, aggregation logic, and routing decisions. If each repository needs its own local discussion, install the built-in workflow there too, but treat the central rollup as the broader portfolio view.

### Enterprise-wide: extend the central aggregation pattern

For multiple organizations, use one or more control-plane repositories that aggregate across repository groups, business units, or organizations.

```aw wrap
---
on:
  schedule: weekly on monday around 10:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  discussions: read
engine: copilot
tools:
  github:
    github-token: ${{ secrets.GH_AW_ENTERPRISE_READ_TOKEN }}
    toolsets: [repos, orgs]
  bash:
    - "gh aw logs *"
safe-outputs:
  create-discussion:
    max: 1
---

# Enterprise Observability Rollup

Collect normalized observability data across approved organizations and repositories, then publish a portfolio report with shared routing and prioritization.
```

This should be treated as fleet operations. The goal is not to replicate the repository-level workflow everywhere and stitch the output together later. The goal is to keep aggregation, policy, and prioritization in one place.

> [!TIP]
> For org-wide and enterprise-wide deployment, start with a pilot allowlist of repositories before expanding coverage. The central aggregation model is operationally safer when authentication, repo discovery, and rollup logic are still being tuned.

## What it analyzes

Built around `gh aw logs`, the kit prefers `episodes[]` and `edges[]` to analyze orchestrator and worker runs as one logical execution, avoiding misreads of delegated runs in isolation. When episode summaries are insufficient, it audits individual runs to explain regressions or MCP failures. For portfolio review, it uses targeted workflow-file inspection to confirm trigger or schedule overlap.

It consumes signals from `gh aw logs` and `gh aw audit`:

- Episode-level rollups for lineage, risk, blocked requests, MCP failures, and suggested route
- Per-run metrics: duration, action minutes, token usage, turns, warnings, and `estimated_cost`
- Effective Tokens — a normalized metric weighting input, output, cache-read, and cache-write tokens by per-model multiplier, enabling cross-run and cross-model comparisons
- Behavior fingerprints and agentic assessments to distinguish overkill workflows from genuinely agentic ones
- Portfolio signals from repeated overkill assessments, weak activity, instability, and workflow overlap

## Visual report form

The kit can produce a chart-backed report format designed for fast interpretation. Instead of relying only on prose, the discussion can include a `Visual Diagnostics` section with a small number of scientific-style plots that make portfolio and observability signals legible at a glance.

The kit is designed around four fixed plot types, each answering a different maintainer question:

| Chart | What it shows | Key question |
|-------|---------------|--------------|
| **Episode Risk-Cost Frontier** | Episodes in cost-risk space (x=cost, y=risk score derived from risky nodes/MCP failures/blocked requests, size=run count) | Which execution chains sit on the cost-risk frontier? |
| **Workflow Stability Matrix** | Workflow-by-metric heatmap of instability signals (risky run rate, fallback rate, poor-control rate, etc.) | Which workflows are chronically unstable vs. noisy in one dimension? |
| **Repository Portfolio Map** | Scatter by cost/value proxy; quadrants labeled keep/optimize/simplify/review | Which workflows deserve investment, simplification, or a decision? |
| **Workflow Overlap Matrix** | Workflow-by-workflow similarity heatmap (task domain, trigger/schedule, fingerprints) | Which workflows solve the same problem closely enough to justify consolidation? |

Each chart should support a decision: the overlap matrix targets consolidation review, the portfolio map targets prioritization, and the risk-cost frontier targets immediate optimization. Read plots outcome-adjusted — cost per successful run is more useful than raw spend; effective tokens per successful run is more useful than raw token totals.

## Metric glossary

| Metric | Definition |
|--------|-----------|
| `episode_risk_score` | Composite risk score combining risky nodes, poor-control nodes, MCP failures, blocked requests, regression markers, and escalation eligibility |
| `workflow_instability_score` | Workflow-level score from repeated risky runs, poor-control assessments, resource-heavy assessments, fallback usage, and MCP failures — separates chronic instability from one-off incidents |
| `workflow_value_proxy` | Repository-local value proxy (not a business KPI) combining recent successful usage, stability, repeat use, and absence of overkill signals — used to rank workflows into keep/optimize/simplify/review |
| `workflow_overlap_score` | Approximate similarity between two workflows, blending task domain, trigger/schedule similarity, naming, and behavioral fingerprints — supports consolidation review, not proof of duplication |
| `cost per successful run` | Preferred cost view that separates expensive-but-effective workflows from expensive-and-unreliable ones |
| `effective tokens per successful run` | Preferred token-efficiency view across routes and models, accounting for token class weighting and model multipliers |

## Calibration from a real repository sample

A live sample of recent runs in this repository surfaced three calibration lessons:

1. **Sparse cost data**: Effective Tokens carried more usable signal than estimated dollar cost. When cost fields are sparse, switch the portfolio map's x-axis to Effective Tokens per successful run.
2. **Coarse task domains**: Most runs landed in `general_automation`, but behavior fingerprints (directed/exploratory/adaptive) still separated them meaningfully. Compare by behavior cluster when the domain layer is too coarse.
3. **`partially_reducible` is a hint, not a verdict**: Treat it as significant only when paired with high Effective Tokens, high turn counts, or repeated resource-heavy assessments.

## Domain-specific reading

The same signals mean different things across workflow types. Compare within similar domains first — a cheap triage workflow and an expensive research workflow are not substitutes for each other.

| Domain | Key question | Notes |
|--------|--------------|-------|
| `triage`, `repo_maintenance`, `issue_response` | Is the workflow too agentic for its job? | Strongest candidates for deterministic replacements, smaller models, narrow-tool routing |
| `research` | Does repeated cost have evidence of value? | Consider moving data-gathering into deterministic pre-steps; keep agent for the analytical core |
| `code_fix` | Does cost combine with instability, blocked requests, or weak control? | Higher cost is acceptable when write actions are intentional and controlled |
| `release_ops` | Is the workflow stable and repeatable? | Reliability dominates; moderate cost is acceptable, repeated instability is not |
| Delegated workflows | Is the worker justified within its episode chain? | A worker expensive in isolation may be fine inside a coherent larger execution |

### Report form

The most useful discussion form is:

1. `Executive Summary` for the overall decision.
2. `Key Metrics` for repository-level scale.
3. `Highest Risk Episodes` and `Episode Regressions` for operational findings.
4. `Visual Diagnostics` with the four charts in the fixed order above.
5. `Portfolio Opportunities` for repository-level cleanup candidates.
6. `Recommended Actions` for the final ranked decisions.

Under each chart, the report should include two short blocks: `Decision` and `Why it matters`. That keeps the visuals analytical instead of decorative.

## Why it matters for COGS reduction

The kit turns agentic workflow spend into a reviewable operational signal. It surfaces four common sources of waste:

- **Overbuilt workflows**: Resource-heavy runs, repeated `latest_success` comparisons, or overkill assessments signal candidates for smaller models, tighter prompts, or deterministic automation.
- **Avoidable control failures**: Repeated blocked requests, MCP failures, or poor-control assessments mean tokens and Actions minutes are going to retries and fallback paths rather than useful work.
- **Hidden orchestration costs**: Episode rollups expose the true aggregate cost of distributed workflows that dispatch workers or chain `workflow_run` triggers.
- **Low-priority optimization**: Escalation logic groups repeated problems into a single actionable report, so owners focus on the highest-value fixes rather than one issue per workflow.

## Accuracy and cost caveats

The kit is accurate as an observability and optimization tool, but its cost signals are not equivalent to billing records.

`action_minutes` is an estimate derived from workflow duration and rounded to billable minutes. It is useful for relative comparison and trend detection, but it does not represent a GitHub invoice line item.

`estimated_cost` is only as authoritative as the engine logs that produced it. For some engines, the value comes from structured log fields emitted by the runtime. For portfolio analysis and prioritization this is usually sufficient, but the number should still be treated as a run-level estimate rather than finance-grade accounting.

Effective Tokens are also intentionally not a billing unit. They are a normalization layer that makes cross-run and cross-model comparisons more useful. Use them to answer “which workflows are inefficient?” rather than “what exact amount will appear on the invoice?”

## When to use it

This pattern is a good fit when:

- A repository has multiple agentic workflows and maintainers need a weekly operational summary.
- Orchestrated workflows make per-run analysis misleading.
- The team wants an evidence-based way to identify model downgrades, prompt tightening, deterministic replacements, or workflow cleanup candidates.
- The repository already uses `gh aw logs` and `gh aw audit` for investigation and wants the same signals in an automated report.

This pattern is a poor fit when a repository has only one low-frequency workflow or when exact billing reconciliation is the primary requirement.

For organization-wide or enterprise-wide deployment, it is also a poor fit as a direct copy-paste workflow if there is no central repository, no cross-repository token strategy, or no clear allowlist of repositories to observe.

## Relationship to other tools

The kit does not replace the lower-level debugging tools.

- Use [`gh aw logs`](/gh-aw/reference/audit/#gh-aw-logs---format-fmt) to inspect cross-run trends directly.
- Use [`gh aw audit`](/gh-aw/reference/audit/#gh-aw-audit-run-id-or-url-run-id-or-url) for a detailed single-run report.
- Use [Cost Management](/gh-aw/reference/cost-management/) to understand Actions minutes, inference spend, and optimization levers.
- Use [Cross-Repository Operations](/gh-aw/reference/cross-repository/) and [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) when the observability workflow needs to read or coordinate across multiple repositories.

The Agentic Observability Kit sits above those tools. It is the scheduled reviewer that turns those raw signals into one repository-level report.

## Portfolio review capabilities

The standalone `portfolio-analyst` workflow has been superseded by the Agentic Observability Kit. Maintainers who want one weekly report should use the kit instead of running both workflows. The highest-value cleanup candidates are workflows dominated on more than one axis — expensive and unstable, cheap but consistently low-value, or overlapping and weaker than a nearby alternative.

## Source workflow

The built-in workflow lives at [`/.github/workflows/agentic-observability-kit.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/agentic-observability-kit.md).

> [!NOTE]
> The workflow prompt prefers deterministic episode data over prompt-time reconstruction. If episode data is missing or incomplete, the report is expected to call that out as an observability finding rather than silently guessing.
