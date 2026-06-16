---
emoji: 📊
description: Executive impact efficiency report from workflow outcomes tied to tracked objectives.
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
  actions: read
  issues: read
cache:
  - key: objective-impact-report-cache-${{ github.run_id }}
    name: Save objective impact report dataset cache
    path: /tmp/gh-aw/agent/objective-impact-report
    restore-keys: |
      objective-impact-report-cache-
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    read-only: true
    toolsets: [default]
  bash:
    - "cat /tmp/gh-aw/agent/objective-impact-report/*.json"
    - "cat /tmp/gh-aw/agent/objective-impact-report/*.jsonl"
    - "jq *"
    - "ls /tmp/gh-aw/agent/objective-impact-report"
    - "head -n * /tmp/gh-aw/agent/objective-impact-report/*.json /tmp/gh-aw/agent/objective-impact-report/*.jsonl"
pre-agent-steps:
  - name: Prepare deterministic impact datasets
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      EXPR_GITHUB_REPOSITORY: ${{ github.repository }}
    run: bash scripts/prepare-objective-impact-report-dataset.sh
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

## Required Inputs (already precomputed)

Use these deterministic files first:

- /tmp/gh-aw/agent/objective-impact-report/dataset-manifest.json
- /tmp/gh-aw/agent/objective-impact-report/run-context.json
- /tmp/gh-aw/agent/objective-impact-report/objective-mapping.json
- /tmp/gh-aw/agent/objective-impact-report/workflow-logs.json
- /tmp/gh-aw/agent/objective-impact-report/merged-prs-linked.json
- /tmp/gh-aw/agent/objective-impact-report/closed-unmerged-prs-linked.json
- /tmp/gh-aw/agent/objective-impact-report/safe-output-issue-evaluations.jsonl
- /tmp/gh-aw/agent/objective-impact-report/safe-output-issue-summary.json

Do **not** re-fetch these datasets with GitHub tools unless a required file is missing, empty, fails JSON parsing, or the manifest explicitly identifies a required live fallback for a missing field.

## Goal

Produce a comprehensive executive report on what work was performed, what AIC tokens were spent on, which outcomes delivered the highest impact, and which workflows contributed that impact. The report must clearly answer: *What did we build, fix, and ship, what was the most impactful work in the repository, which workflows drove that impact, and was it worth the cost?*

Focus only on **pull request outcomes** and **safe output outcomes** (issues created or closed via the safe-output mechanism). Other outcome types are excluded because their acceptance criteria are not yet well-defined and most remain pending.

Use this model:

```text
Outcome = a PR or safe-output issue produced by a GitHub Agentic Workflow run
Objective Value = numeric value from the repository objective-mapping configuration applied to traced root labels
Outcome Indicator = 1 for accepted/delivered outcomes, 0 otherwise
Outcome Value = Outcome Indicator × Objective Value
Impact Efficiency = Σ Outcome Value / AI Credits
```

Treat AI Credits as total model-credit cost consumed by the workflow runs that produced the analyzed outcomes.
Start with `/tmp/gh-aw/agent/objective-impact-report/workflow-logs.json` and `/tmp/gh-aw/agent/objective-impact-report/dataset-manifest.json` as the source of truth for deterministic run collection.
When available, use deterministic precomputed run data that already includes each run's `aic` field.
Prefer existing gh-aw outputs that already surface `aic`, such as pre-downloaded `gh aw logs --json` data or audit/log artifacts derived from the same run summaries.
Only fall back to MCP or other live retrieval if deterministic precomputed AIC inputs are unavailable or the manifest says the fallback is still required.
Use the same time window for AIC as for outcomes.

Perform direct workflow attribution for every analyzed outcome.
Outcomes deliver value.
Objectives provide context and importance.
Workflows explain where delivered value came from.
AI Credits provide cost.
Do not use an LLM judge.

## AIC Source of Truth

Resolve AI Credits in this order:

1. Deterministic precomputed `/tmp/gh-aw/agent/objective-impact-report/workflow-logs.json` data with per-run `aic`
2. Pre-downloaded audit/log artifacts that already expose run-level `aic`
3. MCP or other live retrieval only as a documented fallback

If a run's `aic` field is missing or null, treat it as `0` and count it as missing-cost data in the report.

## Scope

Analyze only the following outcome types from the last 180 days:

- **Pull request outcomes**: PRs created by GitHub Agentic Workflow runs **that have a linked closing issue** (`Closes #N`). Accepted = merged. Rejected = closed without merge. Skip open (pending) PRs. **Exclude entirely** any PR without a traceable linked issue — do not fall back to PR labels.
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

1. For pull-request outcomes, trace the PR to its linked closing issue (`Closes #N`) and use that root issue's labels.
2. If PR root tracing fails (no linked closing issue found), **exclude the PR from analysis entirely**. Do not fall back to PR labels. Count it in the "PRs excluded (no linked issue)" total.
3. For safe-output issue outcomes, use the precomputed records in /tmp/gh-aw/agent/objective-impact-report/safe-output-issue-evaluations.jsonl as the primary source for outcome state, workflow attribution, and issue identity, then use labels on the safe-output issue itself.
4. Record the traced root URL when one is found so the report preserves an audit trail.
5. If no mapped objective labels can be found after tracing, mark the outcome as `unmapped`, exclude it from `Σ Outcome Value`, and report it separately.

Use `/tmp/gh-aw/agent/objective-impact-report/merged-prs-linked.json` and `/tmp/gh-aw/agent/objective-impact-report/closed-unmerged-prs-linked.json` as the primary deterministic PR dataset for discovering linked closing issues before making any live GitHub queries.

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

Also compute per-workflow attribution using the workflow run that directly produced each analyzed outcome:

```text
Workflow Contributed Value = sum(Outcome Value for outcomes produced by that workflow)
Workflow Accepted Outcomes = count(accepted, mapped outcomes produced by that workflow)
Workflow AI Credits        = sum(run.aic for analyzed runs of that workflow)
Workflow Value Share       = Workflow Contributed Value / Total Outcome Value
Workflow Impact Efficiency = Workflow Contributed Value / Workflow AI Credits
```

Use the workflow name from the producing run as the attribution key. If multiple runs from the same workflow produced analyzed outcomes, aggregate them together. If the workflow name cannot be resolved for an analyzed outcome, report it as unattributed and exclude it from workflow ranking totals while still counting the outcome in overall delivered-value totals.

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

### Executive Summary

Write 2–4 sentences that directly answer: *What did we work on, what was the highest-impact work, which workflows contributed most to that impact, and how efficiently were AIC tokens spent?* Highlight the most impactful objective categories, the workflows contributing the most value, and any significant gaps (e.g., large AIC spend with no mapped objective value).

### Summary

| Metric | PRs | Safe-output Issues | Combined |
|---|---:|---:|---:|

Include:

- PRs analyzed with linked issue (merged / closed / excluded open)
- PRs excluded (no linked closing issue)
- Safe output outcomes analyzed
- Outcomes mapped to objectives
- Unmapped outcomes
- Accepted outcome count
- Total outcome value
- AI Credits
- Impact Efficiency

### What We Worked On

Group all **accepted, mapped** outcomes by objective category (the highest-value objective label from the mapping). For each category, list:

- Objective category name and its mapping value
- Number of accepted outcomes in this category
- Total outcome value contributed
- AIC consumed by outcomes in this category
- Impact Efficiency for this category (total outcome value / AIC consumed)
- Representative examples (up to 3 linked outcomes)

Sort categories by total outcome value descending. Also call out separately which category consumed the **most AIC** (highest denominator cost), so readers can see where budget was spent regardless of value delivered.

This section should make the most impactful work in the repository obvious at a glance.

### Which Workflows Drove That Impact

Group all analyzed outcomes by the workflow that directly produced them. For each workflow, list:

- Workflow name
- Number of accepted, mapped outcomes attributed to this workflow
- Total outcome value contributed
- Share of total delivered outcome value
- AIC consumed by this workflow's analyzed runs
- Workflow Impact Efficiency (contributed value / AIC consumed)
- Top objective categories this workflow contributed to
- Representative examples (up to 3 linked outcomes)

Sort workflows by total outcome value descending. Also call out separately:

- which workflow contributed the **most total value**
- which workflow contributed the **largest share** of delivered value
- which workflow consumed the **most AIC**

If any analyzed outcomes cannot be attributed to a workflow, report an unattributed bucket with counts and total outcome value, but do not rank it alongside named workflows.

### Top outcomes by outcome value

| Outcome | Workflow | Type | Root / Associated Objective | Objective Value | Outcome Value |
|---|---|---|---|---:|---:|

List the top 15 outcomes with highest Outcome Value. Include a link to the PR or issue.

### Unmapped outcomes

| Outcome | Type | Reason objective was not mapped |
|---|---|---|

Only include outcomes that were in scope (linked-issue PRs and safe-output issues) but had no matching label in the objective mapping. Do not include PRs that were excluded for lacking a linked issue — those are already counted in "PRs excluded".

### Interpretation

Compare:

- accepted outcome count alone
- Impact Efficiency

Explain which one better reflects meaningful delivered value relative to cost.

Call out the most significant findings:

- which objective category delivered the most value per AIC
- which workflow contributed the most delivered value
- which workflows consumed AIC with little or no mapped value

### Data quality

Mention missing or weak links in:

- PR root tracing and linked-closing-issue coverage (count of PRs excluded for lacking a linked issue)
- safe-output issue label mapping coverage in `.github/objective-mapping.json`
- workflow attribution coverage and any unattributed analyzed outcomes
- AI Credits availability

State whether AI Credits came from deterministic precomputed data or from a live fallback path.

If AI Credits are unavailable, still produce the delivered-value analysis and clearly state that the cost-normalized Impact Efficiency metric could not be computed.

## Safe output

Use only `close-issue` and `create-issue`.
