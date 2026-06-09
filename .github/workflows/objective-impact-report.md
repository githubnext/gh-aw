---
emoji: 📊
description: Impact efficiency report from workflow outcomes and linked objectives.
on:
  workflow_dispatch:
permissions:
  issues: read
safe-outputs:
  create-issue:
    max: 1
---

# Impact Efficiency Report

## Goal

Test whether Impact Efficiency is a more meaningful signal than accepted outcome counts alone.

Use this model:

```text
Outcome = recorded work item produced by a GitHub Agentic Workflow run
Objective = issue/epic/work item linked to the outcome
Objective Value = value from planning metadata (priority, severity, milestone, project)
Outcome Indicator = 1 for accepted/delivered outcomes, 0 otherwise
Outcome Value = Outcome Indicator × Objective Value
Impact Efficiency = Σ Outcome Value / AI Credits
```

Treat an outcome as one recorded result item produced by a GitHub Agentic Workflow run (for example, a PR change, completed fix, or report action), which may later be accepted or not accepted.
Use workflow run outputs/artifacts and linked GitHub objects (issues, PRs, comments, discussions) as the outcome source of truth.
Treat AI Credits as total model-credit cost consumed by the workflow runs that produced the analyzed outcomes.
Retrieve AI Credits from workflow-run usage/billing data available to the run context, and use the same time window as outcomes.

Do not perform workflow attribution.
Outcomes deliver value.
Objectives provide context and importance.
AI Credits provide cost.
Do not use an LLM judge.

## Scope

Analyze workflow outcomes and linked objectives from the last 180 days.

## Objective value mapping

For each outcome, find the associated objective first, then compute `Objective Value`.

Use the first matching priority or severity signal from objective labels or fields as the base value:

```text
P0 / urgent / critical = 100
P1 / high              = 50
P2 / medium            = 20
P3 / low               = 5
unknown                = 1
```

Recognize common label forms case-insensitively:

```text
P0, priority:P0, priority/P0, severity:critical, critical, urgent
P1, priority:P1, priority/P1, severity:high, high
P2, priority:P2, priority/P2, severity:medium, medium
P3, priority:P3, priority/P3, severity:low, low
```

Then apply planning context adjustments:

```text
Objective is assigned to a milestone = +10
Objective is assigned to a project   = +10
```

Cap `Objective Value` at 120 (`100 + 10 + 10` maximum from base plus planning adjustments).

The cap prevents a small number of heavily tagged objectives from dominating the metric.

All other labels are classification only.

## Outcome association rules

For each workflow outcome, associate one objective using this order:

1. Explicit linked issue or work item reference in the outcome.
2. Issue linked through the related pull request.
3. Parent issue/epic if explicitly linked.
4. If no objective can be found, mark as `unmapped`, exclude it from `Σ Outcome Value`, and report it separately.

## Computation

For each outcome:

```text
Outcome Indicator = 1 for accepted/delivered outcome, 0 for rejected, abandoned, or incomplete outcome
Outcome Value = Outcome Indicator × Objective Value
```

Treat pending-review outcomes as `Outcome Indicator = 0` until explicitly accepted.

Accepted/delivered outcome means the intended result was accepted in GitHub state (for example: merged PR, closed issue with completion signal, or explicit accepted status in the workflow outcome record).

Then compute:

```text
Accepted Outcome Count = count(outcomes where Outcome Indicator = 1)
Total Outcome Value    = sum(Outcome Value)
Impact Efficiency      = Total Outcome Value / AI Credits  (value units per AI Credit; undefined when AI Credits = 0)
```

If AI Credits is missing or zero, report that Impact Efficiency is not computable and explain whether credits data was unavailable or no credits were consumed in the analysis window.

## Report

Create one issue titled:

```text
Impact Efficiency Report - YYYY-MM-DD
```

The report must include:

### Summary

- Outcomes analyzed
- Objectives mapped
- Unmapped outcomes
- Accepted outcome count
- Total outcome value
- AI Credits
- Impact Efficiency

### Top outcomes by outcome value

| Outcome | Associated objective | Objective value signals | Objective Value | Outcome Value |
|---|---|---|---:|---:|

### Top objectives by delivered value

| Objective | Priority/Severity | Milestone | Project | Delivered Outcome Value |
|---|---|---|---|---:|

### Unmapped outcomes

| Outcome | Reason objective was not mapped |
|---|---|

### Interpretation

Compare:

- accepted outcome count alone
- Impact Efficiency

Explain which one better reflects meaningful delivered value relative to cost.

### Data quality

Mention missing or weak links in:

- outcome-to-objective association
- priority/severity metadata
- milestone/project metadata
- AI Credits availability

## Safe output

Use only `create-issue`.

If a report for today already exists, do nothing.
