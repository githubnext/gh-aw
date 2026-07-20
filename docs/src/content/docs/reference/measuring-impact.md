---
title: Measuring Impact
description: Learn how to measure the impact of agentic workflows using cost, outcomes, funnel metrics, and system-level trends.
sidebar:
  order: 295
---

Measure impact by using **early cost signals** alongside
**later outcome signals**.

Do not try to collapse them into a single score.

Use this page to choose metrics and interpret them together. For
spend controls, see [Cost management](/gh-aw/reference/cost-management/).
For downstream result tracking, see [Outcomes](/gh-aw/reference/outcomes/).

## Timing of Cost and Outcomes

Cost estimates are usually available early, while accurate cost
measurement often arrives later. In practice, cost is built from
inputs such as GitHub Actions minutes, runner duration, inference
usage, and storage or artifact retention.

Outcomes are often delayed and downstream. A comment may only get
a response days later, a proposed change may only matter once it
is accepted or merged, and a created issue may only create value
once it is resolved.

If you collect outcomes in a separate workflow or reporting pass,
that delay is expected.

The right default is not a custom impact formula. Use a small set
of metrics that gh-aw already exposes and read them on the
timeline where they become trustworthy.

Use a small set of metric layers instead of one synthetic score.
Operational metrics tell you whether the workflow runs reliably;
cost-efficiency metrics tell you what useful execution costs;
outcome metrics tell you whether the workflow produced something
that mattered; and long-term impact metrics tell you whether the
workflow improved the broader system.

In practice, that usually means starting with run count,
completion rate, retries, duration, cost per successful run,
useful output rate, acceptance rate, and time to adoption.

Long-term impact matters, but it is usually the hardest to measure
and attribute directly.

For most teams, the best starting point is a small set of direct
metrics: run volume, execution success, Actions minutes,
inference cost, useful output rate, and acceptance over time.
These are easy to observe and easy to improve. Once they are
stable, connect them to downstream questions such as whether
outputs were used, accepted, merged, or helped reduce later work.

## A Practical Measurement Model

Start simple. For most teams, the right first dashboard is run
volume, execution success, cost, useful output rate, and
acceptance over time. That is enough to tell you whether a
workflow is doing useful work at a reasonable price.

Use the built-in telemetry before designing anything more complex.
`gh aw logs` gives you the run and cost side. The
[Outcomes](/gh-aw/reference/outcomes/) model gives you the
downstream acceptance side. If you need repository-wide or
organization-wide trends, send the same data to
[OpenTelemetry](/gh-aw/guides/open-telemetry/).

This is enough to show where impact is breaking down: a workflow
that runs reliably but produces little of value, one that creates
useful output that rarely gets adopted, or one that is effective
but too expensive. Detailed downstream outcome evaluation belongs
in [Outcomes](/gh-aw/reference/outcomes/).

A workflow can look efficient on its own while still reducing total
system value. That usually happens when two workflows act on the
same issue or pull request, overlapping triggers create duplicate
outputs, or competing suggestions increase review burden.

Measure both local efficiency and system-level overlap. The useful
questions are whether multiple workflows are acting on the same
event type, how often they produce duplicate outputs, what the
cost per unique accepted outcome looks like across the system,
and which workflows add unique value rather than repeating other
automation.

Teams SHOULD define explicit acceptance thresholds before deploying
a workflow to production. The following starting-point thresholds are
a baseline recommendation — adjust them based on your workflow's task type,
frequency, and business context:

| Metric | Starting-point threshold | Notes |
|---|---|---|
| Completion rate | ≥ 90% | Runs that complete all steps and exit without an unhandled error (exit code 0) |
| Useful output rate | ≥ 70% | Runs that produce at least one accepted safe output |
| Acceptance rate | ≥ 50% | Accepted outcomes divided by total safe outputs produced |
| Cost per accepted outcome | Establish a baseline in the first two weeks, then alert on 2x regression | Use AIC from `gh aw logs` divided by accepted outcome count |
| Retry rate | ≤ 10% | Runs retried due to failure |

These are baseline recommendations, not hard limits. A triage
workflow operating at 40% acceptance may still be net-positive if
the accepted cases save significant reviewer time. Document the
chosen thresholds in a `MONITORING.md` or in workflow frontmatter
comments so the team can revisit them as the workflow matures.

> [!NOTE]
> For spend controls and AIC budget configuration, see
> [Cost Management](/gh-aw/reference/cost-management/). For
> downstream result tracking and outcome state definitions, see
> [Outcomes](/gh-aw/reference/outcomes/).

## System Overlap and Waste

Waste is any cost, time, or reviewer attention that does not
produce proportional value. Common sources include redundant
runs, duplicate outputs, repeated context collection, expensive
model calls for deterministic work, and outputs with consistently
low usage or acceptance.

Typical fixes include consolidating overlapping workflows,
sharing intermediate artifacts, caching stable context, and
moving deterministic work out of the agent path.

Do not overreact to single numbers. Trend data is usually more
useful. Look for cost per successful run moving down, useful
output rate and acceptance moving up, retries dropping, and
system overlap decreasing.

## Related Documentation

See [Cost Management](/gh-aw/reference/cost-management/) for spend
controls and AIC budget configuration — including failure safeguards
when pricing data is unavailable. See [Outcomes](/gh-aw/reference/outcomes/)
for downstream result tracking, acceptance criteria, and outcome state
definitions.