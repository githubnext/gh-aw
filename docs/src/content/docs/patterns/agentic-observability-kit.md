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

The built-in workflow is repository-scoped. It reads workflow runs for the repository where it is installed and produces one report for that repository.

At repository scope, the report combines two layers:

- operational observability for recent runs, episodes, regressions, and control failures
- an evidence-based repository portfolio appendix for overlap, stale workflows, and weakly justified agentic workflows

The same pattern can be extended to an organization. For org-wide reporting, run the workflow from a central repository or control-plane repository, give it cross-repository read access, and aggregate results across target repositories using the [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) or [CentralRepoOps](/gh-aw/patterns/central-repo-ops/) patterns. In practice, that means collecting `gh aw logs` or MCP `logs` output per repository, then generating an organization-level rollup.

At organization or enterprise scope, the same pattern can grow into a broader portfolio-governance rollup. That is where deeper questions such as consolidation, duplicate ownership, shared policy, and fleet-wide prioritization fit best.

Enterprise-wide use is possible as an architecture, but it is not a single built-in turnkey workflow today. At enterprise scope, the usual design is a control-plane repository that fans out across multiple organizations or repository groups, collects normalized run data, and publishes a portfolio report. This is closer to a fleet-operations pattern than a drop-in single-repository workflow.

> [!IMPORTANT]
> The built-in Agentic Observability Kit should be treated as the repository-level building block. Organization-wide and enterprise-wide deployments require additional cross-repository authentication, central orchestration, and portfolio aggregation logic.

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

The kit is built around the deterministic lineage data returned by `gh aw logs`. Instead of treating every workflow run as an isolated event, it prefers `episodes[]` and `edges[]` so orchestrator and worker runs are analyzed as one logical execution. This avoids misreading delegated runs in isolation and makes cost, risk, and control signals easier to attribute.

The episode rollups include aggregate fields such as total runs, total tokens, total estimated cost, blocked requests, MCP failures, risky nodes, and a suggested routing hint for follow-up. When the summary data is not sufficient, the workflow can audit a small number of individual runs to explain the latest regression, a new MCP failure, or a changed write posture.

For the portfolio portion of the report, the kit can also use targeted workflow-file inspection to confirm trigger or schedule overlap when recent run data suggests that two workflows may be serving the same job.

## What it computes

The kit consumes several classes of signals that are already produced by `gh aw logs` and `gh aw audit`:

- Episode-level rollups for lineage, risk, blocked requests, MCP failures, and suggested route.
- Per-run metrics such as duration, action minutes, token usage, turns, warnings, and `estimated_cost`.
- Effective Tokens, a normalized token metric that weights input, output, cache-read, and cache-write tokens before applying a per-model multiplier.
- Behavior fingerprint and agentic assessments, which help distinguish overkill workflows from genuinely agentic ones.
- Repository-level portfolio signals derived from repeated overkill assessments, weak recent activity, repeated instability, and possible overlap in workflow purpose, trigger pattern, or schedule.

Effective Tokens matter because raw token counts alone are not comparable across models or token classes. The implementation normalizes token classes first, then applies a model multiplier. This makes it easier to compare a cache-heavy run against an output-heavy run, or a lightweight model against a more expensive one, without collapsing everything into raw token totals.

## Visual report form

The kit can produce a chart-backed report format designed for fast interpretation. Instead of relying only on prose, the discussion can include a `Visual Diagnostics` section with a small number of scientific-style plots that make portfolio and observability signals legible at a glance.

The visual report is most useful when it is concrete. The kit is designed around four fixed plot types, each answering a different maintainer question.

### 1. Episode Risk-Cost Frontier

This plot shows execution episodes in cost-risk space. The x-axis is episode cost, the y-axis is an episode risk score derived from risky nodes, poor-control signals, MCP failures, blocked requests, and escalation eligibility, and the point size reflects run count.

This is the fastest way to answer: which execution chains sit on the cost-risk frontier and deserve immediate attention?

### 2. Workflow Stability Matrix

This plot is a workflow-by-metric heatmap. Each row is a workflow, and the columns represent repeated instability signals such as risky run rate, latest-success fallback rate, resource-heavy assessment rate, poor-control rate, blocked-request incidence, and MCP failure incidence.

This is the fastest way to answer: which workflows are chronically unstable, and which ones are noisy only in one dimension?

### 3. Repository Portfolio Map

This plot is a workflow portfolio scatter plot. The x-axis represents recent cost, the y-axis represents an evidence-based value proxy, the point size reflects run count, and the quadrants separate workflows into `keep`, `optimize`, `simplify`, and `review`.

This is the fastest way to answer: which workflows deserve investment, which should be simplified, and which demand a maintainer decision?

### 4. Workflow Overlap Matrix

This plot is a workflow-by-workflow similarity heatmap. It is intended to serve the role that people often imagine for a Venn diagram, but in a form that scales beyond two or three workflows. Similarity can be inferred from task domains, naming, trigger or schedule similarity, behavioral fingerprints, and repeated overlap signals from recent runs.

This is the fastest way to answer: which workflows may be solving the same problem closely enough to justify consolidation review?

This format is better than a purely textual report when maintainers need to answer questions such as:

- which workflows sit on the cost-risk frontier
- which workflows are stable but overbuilt
- which workflows cluster together strongly enough to deserve consolidation review
- which problems are isolated incidents versus repeated patterns across the repository

The goal is not visual novelty for its own sake. Each chart should support a decision. In practice, the overlap matrix supports consolidation review, the portfolio map supports prioritization, and the risk-cost frontier supports immediate optimization work.

The most useful reading of those plots is outcome-adjusted rather than usage-only. Cost is more informative when read as cost per successful run. Token volume is more informative when read as effective tokens per successful run. Tool overhead is more informative when read as median tool calls or turns per successful run rather than raw totals.

## Metric glossary

The visual report uses a small number of derived metrics to keep the plots decision-oriented.

`episode_risk_score`
This is a composite risk score for a single execution episode. It combines risky nodes, poor-control nodes, MCP failures, blocked requests, repeated regression markers, and escalation eligibility. It exists to answer one question quickly: which episodes combine multiple warning signals at once?

`workflow_instability_score`
This is a workflow-level instability score derived from repeated risky runs, poor-control assessments, resource-heavy assessments, latest-success fallback usage, blocked requests, and MCP failures. It exists to separate chronic instability from one-off incidents.

`workflow_value_proxy`
This is a repository-local proxy for workflow value. It is not a business KPI. It combines successful recent usage, stability, repeat use, and the absence of strong overkill or reduction signals. It exists to help rank workflows into `keep`, `optimize`, `simplify`, and `review` rather than to claim objective business value.

`workflow_overlap_score`
This is an approximate similarity score between two workflows. It blends task domain, trigger or schedule similarity, naming, behavioral fingerprints, and assessment patterns. It exists to support consolidation review, not to prove duplication with mathematical certainty.

`cost per successful run`
This is the preferred cost view when enough successful runs exist. It is more decision-useful than raw spend because it separates expensive-but-effective workflows from expensive-and-unreliable workflows.

`effective tokens per successful run`
This is the preferred token-efficiency view when comparing routes or workflows across models. It is more useful than raw token totals because it accounts for token class weighting and model differences.

## Calibration from a real repository sample

In this repository, a live sample of recent runs produced three practical calibration lessons for the kit.

First, estimated dollar cost was sparse. Effective Tokens carried much more of the usable efficiency signal across workflows. In repositories with similar telemetry, the portfolio map should switch its primary x-axis from recent cost to Effective Tokens, or Effective Tokens per successful run when enough successful runs exist.

Second, task-domain coverage was coarse. Most sampled runs landed in `general_automation`, while the behavior fingerprints still separated them into meaningful groups such as directed lean runs, exploratory heavy runs, and adaptive moderate runs. In repositories with similar distributions, the portfolio analysis should compare workflows by behavior cluster and workflow family when the domain layer is too coarse to be reliable.

Third, `partially_reducible` appeared often enough that it should be treated as a repeated reduction hint, not an automatic negative verdict. In practice, that signal becomes most useful when it appears together with high Effective Tokens, high turn counts, or repeated resource-heavy behavior.

Those findings do not replace the general design. They make the default interpretation more robust when repositories have sparse cost fields, weak domain separation, or many exploratory workflows.

## Domain-specific reading

The same signals do not mean the same thing for every workflow type.

For `triage`, `repo_maintenance`, and `issue_response`, the most valuable question is whether the workflow is too agentic for its job. These domains are the strongest candidates for deterministic replacements, smaller models, and narrow-tool routing.

For `research`, broader tool use and exploration can be justified, but repeated cost still needs evidence of value. In practice, the most important question is often whether data-gathering work should move into deterministic pre-steps while the agent keeps only the analytical core.

For `code_fix`, higher cost may be justified when successful write actions are intentional and controlled. The most important question is usually not pure spend, but whether the workflow combines cost with instability, blocked requests, or weak control.

For `release_ops`, reliability dominates. The most important question is whether the workflow is stable, repeatable, and well controlled; moderate cost is often acceptable, repeated instability is not.

For delegated workflows, episode-level interpretation matters more than local workflow-level interpretation. A worker that looks expensive in isolation may still be justified inside a coherent larger execution chain.

This is why the kit should compare workflows within similar task domains first. A cheap triage workflow and an expensive research workflow are not automatically substitutes for each other.

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

The kit is useful for COGS reduction because it turns agentic workflow spend into a reviewable operational signal instead of a vague complaint about “expensive runs”. In practice it helps surface four common sources of waste.

First, it highlights workflows that are too expensive for the task domain they serve. A workflow that is consistently resource-heavy, repeatedly compared against `latest_success`, or marked as overkill for agentic execution is a candidate for a smaller model, tighter prompts, or deterministic automation.

Second, it exposes avoidable control failures. Repeated blocked requests, MCP failures, or poor-control assessments often mean the system is spending tokens and Actions minutes on retries, fallback behavior, or incomplete execution paths rather than useful work.

Third, it makes orchestration costs legible. Episode rollups prevent distributed workflows from hiding their true aggregate cost across many child runs. This is important for repositories that dispatch workers or chain `workflow_run` triggers.

Fourth, it gives maintainers a way to prioritize optimization work. The escalation logic is designed to avoid one issue per workflow. Instead, it groups repeated problems into a single actionable report so repository owners can focus on the highest-value fixes first.

Fifth, it helps maintainers spot workflows that may no longer deserve their current shape. When a workflow is repeatedly overkill for its domain, rarely active, or plausibly overlapping with another workflow, the kit can surface that as a cleanup opportunity before the repository accumulates more operational drag.

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
- Use [`gh aw audit`](/gh-aw/reference/audit/#gh-aw-audit-run-id-or-url) for a detailed single-run report.
- Use [Cost Management](/gh-aw/reference/cost-management/) to understand Actions minutes, inference spend, and optimization levers.
- Use [Cross-Repository Operations](/gh-aw/reference/cross-repository/) and [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) when the observability workflow needs to read or coordinate across multiple repositories.

The Agentic Observability Kit sits above those tools. It is the scheduled reviewer that turns those raw signals into one repository-level report.

## Portfolio review capabilities

The standalone `portfolio-analyst` workflow has been superseded by the Agentic Observability Kit. Maintainers who want one weekly report should use the kit instead of running both workflows. The highest-value cleanup candidates are workflows dominated on more than one axis — expensive and unstable, cheap but consistently low-value, or overlapping and weaker than a nearby alternative.

## Source workflow

The built-in workflow lives at [`/.github/workflows/agentic-observability-kit.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/agentic-observability-kit.md).

> [!NOTE]
> The workflow prompt prefers deterministic episode data over prompt-time reconstruction. If episode data is missing or incomplete, the report is expected to call that out as an observability finding rather than silently guessing.
