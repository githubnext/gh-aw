---
emoji: 📊
description: Impact efficiency report from workflow outcomes and linked objectives.
on:
  workflow_dispatch:
permissions:
  issues: read
safe-outputs:
  close-issue:
    required-title-prefix: "Impact Efficiency Report - "
    target: "*"
    max: 1
  create-issue:
    title-prefix: "Impact Efficiency Report - "
    max: 1
---

# Impact Efficiency Report

## Goal

POC: Test whether Impact Efficiency is a more meaningful signal than accepted outcome counts alone.

Focus only on **pull request outcomes** and **safe output outcomes** (issues created or closed via the safe-output mechanism). Other outcome types are excluded from this POC because their acceptance criteria are not yet well-defined and most remain pending.

Use this model:

```text
Outcome = a PR or safe-output issue produced by a GitHub Agentic Workflow run
Objective Value = numeric value from the repository objective-mapping configuration applied to traced root labels
Outcome Indicator = 1 for accepted/delivered outcomes, 0 otherwise
Outcome Value = Outcome Indicator × Objective Value
Impact Efficiency = Σ Outcome Value / AI Credits
```

Treat AI Credits as total model-credit cost consumed by the workflow runs that produced the analyzed outcomes.
When available, use deterministic precomputed run data that already includes each run's `aic` field.
Prefer existing gh-aw outputs that already surface `aic`, such as pre-downloaded `gh aw logs --json` data or audit/log artifacts derived from the same run summaries.
Only fall back to MCP or other live retrieval if deterministic precomputed AIC inputs are unavailable.
Use the same time window for AIC as for outcomes.

Do not perform workflow attribution.
Outcomes deliver value.
Objectives provide context and importance.
AI Credits provide cost.
Do not use an LLM judge.

## AIC Source of Truth

Resolve AI Credits in this order:

1. Deterministic precomputed `gh aw logs --json` style workflow-run data with per-run `aic`
2. Pre-downloaded audit/log artifacts that already expose run-level `aic`
3. MCP or other live retrieval only as a documented fallback

If a run's `aic` field is missing or null, treat it as `0` and count it as missing-cost data in the report.

## Scope

Analyze only the following outcome types from the last 180 days:

- **Pull request outcomes**: PRs created by GitHub Agentic Workflow runs. Accepted = merged. Rejected = closed without merge. Skip open (pending) PRs.
- **Safe output outcomes**: issues created or closed by workflow runs via the safe-output mechanism. Accepted = issue successfully created/closed. Skip any with unresolved state.

Exclude all other outcome types (direct issue outcomes, comments, discussions, etc.). These are omitted because their acceptance criteria are incomplete and most are left pending, which would distort the metric.

## Objective value mapping

For each outcome, find the associated objective first, then compute `Objective Value`.

Use the repository objective mapping as the source of truth:

```text
.github/objective-mapping.json
or OBJECTIVE_MAPPING_JSON when explicitly provided
```

Treat labels on the traced root object as the input to the mapping.
The mapping is label-based and already defines both value and multi-label behavior.

```text
Objective Value = mapping.ComputeObjectiveValue(root_labels)
Objective Labels = mapping.GetObjectiveLabels(root_labels)
```

Do not invent fallback scoring rules such as milestone bonuses, project bonuses, or priority-to-points heuristics when the mapping file is present.

If a traced root object has no labels that exist in the mapping, mark the outcome as `unmapped`.

## Outcome association rules

For each in-scope outcome, follow the implemented root-tracing behavior:

1. For pull-request outcomes, trace the PR to its linked closing issue and use that root issue's labels.
2. If PR root tracing fails, use labels on the PR itself.
3. For safe-output issue outcomes, use labels on the safe-output issue itself.
4. Record the traced root URL when one is found so the report preserves an audit trail.
5. If no mapped objective labels can be found, mark the outcome as `unmapped`, exclude it from `Σ Outcome Value`, and report it separately.

## Computation

For each in-scope outcome:

```text
Outcome Indicator:
  PR outcome:           1 if merged, 0 if closed without merge (open PRs excluded)
  Safe output outcome:  1 if successfully created/closed, 0 otherwise
Outcome Value = Outcome Indicator × Objective Value
```

Then compute:

```text
Accepted Outcome Count = count(outcomes where Outcome Indicator = 1)
Total Outcome Value    = sum(Outcome Value)
AI Credits             = sum(run.aic across analyzed runs)
Impact Efficiency      = Total Outcome Value / AI Credits  (value units per AI Credit; undefined when AI Credits = 0)
```

If AI Credits is missing or zero, report that Impact Efficiency is not computable and explain whether credits data was unavailable or no credits were consumed in the analysis window.
If only some runs are missing `aic`, still compute the metric from the available values and explicitly report how many runs had missing cost data.

## Report

Before creating the new report, search for an existing open issue titled:

```text
Impact Efficiency Report - YYYY-MM-DD
```

If one already exists for today:

1. Close that issue first with a brief comment explaining that it is being replaced by a freshly generated report for the same day.
2. Then create the new report issue.

Create one issue titled:

```text
Impact Efficiency Report - YYYY-MM-DD
```

The report must include:

### Summary

- PR outcomes analyzed (merged / closed / excluded open)
- Safe output outcomes analyzed
- Objectives mapped
- Unmapped outcomes
- Accepted outcome count
- Total outcome value
- AI Credits
- Impact Efficiency

### Top outcomes by outcome value

| Outcome | Type | Associated objective | Objective Value | Outcome Value |
|---|---|---|---:|---:|

### Unmapped outcomes

| Outcome | Type | Reason objective was not mapped |
|---|---|---|

### Interpretation

Compare:

- accepted outcome count alone
- Impact Efficiency

Explain which one better reflects meaningful delivered value relative to cost.

### Data quality

Mention missing or weak links in:

- PR root tracing and linked-closing-issue coverage
- safe-output issue label mapping coverage in `.github/objective-mapping.json`
- AI Credits availability

State whether AI Credits came from deterministic precomputed data or from a live fallback path.

If AI Credits are unavailable, still produce the delivered-value analysis and clearly state that the cost-normalized Impact Efficiency metric could not be computed.

## Safe output

Use only `close-issue` and `create-issue`.
