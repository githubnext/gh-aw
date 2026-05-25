---
title: Outcomes
description: Reference for outcome concepts in GitHub Agentic Workflows, including accepted outcomes, outcome states, and how outcome data relates to cost and observability.
sidebar:
  order: 297
---

Outcomes describe what happened after a [safe output](/gh-aw/reference/safe-outputs/) landed in a repository. Safe outputs record what a workflow did. Outcomes record the repository state that can be observed afterward.

For example, a pull request can be merged or closed, an issue can remain relevant or be dismissed, and a comment can lead to follow-up activity or be ignored. Outcome data is based on repository state, not on the workflow's self-assessment.

This page defines the common outcome states, summarizes what `accepted` means across safe output types, and lists the telemetry and cost rollups built from that data.

## Outcome Efficiency

Token and cost data are necessary, but they are not enough. A workflow can become cheaper because it became more efficient, or because it simply did less useful work. Outcomes make that difference visible by relating effective tokens to accepted results.

Outcome efficiency is measured as effective tokens divided by accepted outcomes. Lower is better: a lower value means the workflow spent less effective AI work per accepted result.

## Outcome States

To support that measurement, every evaluated output is classified into an outcome state. These states provide the base vocabulary for the rest of the page.

| Outcome | Meaning |
| --- | --- |
| `accepted` | The result was kept, merged, completed, or otherwise accepted by the repository state. |
| `rejected` | The result was explicitly undone, closed, removed, or not accepted. |
| `pending` | The result exists, but has not reached a terminal state yet. |
| `ignored` | The result received no meaningful follow-up within the evaluation window. |
| `noop` | The output type is intentionally non-actionable for outcome measurement. |

Some evaluation systems also distinguish lifecycle-oriented states such as bot-driven cleanup or closure. This page keeps the top-level reference to the common states; see the safe-output outcome specification for the extended lifecycle details.

## Accepted Outcomes

An accepted outcome is the simplest useful unit for measuring workflow effectiveness. Typical examples include merged pull requests, issues that remained relevant and were completed, and labels or comments that stuck and were acted on.

Accepted outcomes are intentionally simpler than a full value model. They do not try to rank one accepted result as inherently more important than another.

> [!NOTE]
> Different output types can have different practical importance. The outcomes model keeps the base measurement simple first. If needed, compare workflows within the same output class before introducing more complex weighting.

The table below is the quick lookup for what `accepted` currently means for each safe output type and whether that meaning comes from a dedicated rule, a fallback rule, a limited check, or no implemented rule yet.

Rows marked `fallback rule` use a generic existence check, not a type-specific rule. For exact rules, edge cases, and conformance details, see [Safe Output Outcome Evaluation Specification](https://github.com/github/gh-aw/blob/main/specs/safe-output-outcome-evaluation.md).

Outcome evaluation is based on visible repository state and visible actor identity. A non-bot actor may still be AI-assisted; the lookup reflects what the system can observe, not hidden authoring provenance.

| Safe output type | `accepted` at a glance | Current rule source |
| --- | --- | --- |
| `create_pull_request` | merged | dedicated rule |
| `create_issue` | completed/closed | dedicated rule |
| `add_comment` | reacted to or replied to | dedicated rule |
| `add_labels` | label retention | limited check |
| `add_reviewer` | review target exists | fallback rule |
| `update_issue` | issue still exists | fallback rule |
| `update_pull_request` | PR still exists | fallback rule |
| `close_issue` | still closed | dedicated rule |
| `close_pull_request` | still closed | dedicated rule |
| `close_discussion` | none yet | no implemented rule yet |
| `create_discussion` | none yet | no implemented rule yet |
| `update_discussion` | discussion target exists | fallback rule |
| `create_pull_request_review_comment` | none yet | no implemented rule yet |
| `submit_pull_request_review` | PR still exists | fallback rule |
| `reply_to_pull_request_review_comment` | review target exists | fallback rule |
| `resolve_pull_request_review_thread` | none yet | no implemented rule yet |
| `push_to_pull_request_branch` | merged | dedicated rule |
| `mark_pull_request_as_ready_for_review` | reviewed | dedicated rule |
| `assign_to_agent` | merged or completed | dedicated rule |
| `dispatch_workflow` | dispatch target exists | fallback rule |
| `autofix_code_scanning_alert` | alert target exists | fallback rule |
| `create_code_scanning_alert` | alert target exists | fallback rule |
| `link_sub_issue` | sub-issue link target exists | fallback rule |
| `hide_comment` | none yet | no implemented rule yet |
| `assign_milestone` | milestone still set | dedicated rule |
| `update_project` | project target exists | fallback rule |
| `update_release` | release target exists | fallback rule |
| `noop` | skipped | skipped |
| `missing_tool` | skipped | skipped |

## Telemetry

Outcome data is derived from safe outputs and later checked against repository state. The system records the safe output produced by the workflow, looks up the affected repository object later, and classifies the observed state into an outcome.

This makes outcome evaluation external and observable. The workflow does not decide whether it succeeded; the repository state does.

Outcome information appears in OpenTelemetry spans and related artifacts. Workflow-level rollups such as accepted counts and acceptance rate are emitted on outcome summary or conclusion spans, and per-item spans can carry more detailed fields such as object type, URL, comments, review activity, and zero-touch acceptance.

For the span-level attribute inventory, see [OpenTelemetry](/gh-aw/reference/open-telemetry/).

## Cost and Rollups

Outcomes are most useful when read together with cost data. At the workflow level, the basic questions are how many effective tokens a workflow spent, how many accepted outcomes it produced, and how many effective tokens each accepted outcome cost.

The basic dashboard for outcomes is therefore intentionally small: total effective tokens, total accepted outcomes, effective tokens per accepted outcome, a trend over time, and a workflow ranking by effective tokens per accepted outcome.

For simple workflows, a single run is usually the right unit for outcome measurement.

For orchestrated workflows, multiple runs can belong to one logical execution. In that case, the more meaningful unit is the episode. Outcome and cost totals can be rolled up from runs into episodes using simple sums, and then from episodes into workflow totals and repository totals.

The outcomes model is deliberately narrow. It does not try to estimate the full business value of a workflow, replace human judgment for nuanced quality questions, combine deterministic compute cost and inference cost into one synthetic score, or solve overlap and duplicate-work analysis in the first version.

Those questions may matter later, but they are separate from the base outcomes model described here.

## Related Documentation

- [Cost Management](/gh-aw/reference/cost-management/) explains how workflow cost is measured and reduced.
- [OpenTelemetry](/gh-aw/reference/open-telemetry/) describes the span attributes and artifacts that carry workflow telemetry.
- [Safe Outputs](/gh-aw/reference/safe-outputs/) explains how workflows produce constrained actions.
- [Safe Output Outcome Evaluation Specification](https://github.com/github/gh-aw/blob/main/specs/safe-output-outcome-evaluation.md) defines the detailed evaluation logic for each safe output type.