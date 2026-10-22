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

Token and cost data are necessary, but they are not enough. A workflow can become cheaper because it became more efficient, or because it simply did less useful work. Outcomes make that difference visible by relating AI Credits (AIC) to accepted results.

Outcome efficiency is measured as AI Credits (AIC) divided by accepted outcomes. Lower is better: a lower value means the workflow consumed fewer AI Credits per accepted result.

## Outcome States

To support that measurement, every evaluated output is classified into an outcome state. These states provide the base vocabulary for the rest of the page.

| Outcome | Meaning |
| --- | --- |
| `accepted` | The result was kept, merged, completed, or otherwise accepted by the repository state. |
| `rejected` | The result was explicitly undone, closed, removed, or not accepted. |
| `pending` | The result exists, but has not reached a terminal state yet. |
| `ignored` | The action reached a state without its expected effect, such as a dispatched run skipped without success or failure. |
| `lifecycle` | Closed or removed by the workflow itself as part of its normal operation (for example, a `close-older-issues` workflow) — not a rejection. |
| `lifecycle_close` | A `close_issue` or `close_pull_request` output where the close actor was a lifecycle bot (for example, a stale bot) and no visible non-bot actor has since reopened it. |
| `unknown` | The action cannot be verified: required execution evidence is missing, the evaluator is unsupported, or only target existence is known. |
| `error` | An API or evaluation failure prevented checking the action. |

The CLI uses action-specific classifications,
evidence strengths, and signals. An open issue or PR remains `pending`; approval,
bot comments, and unrelated target state do not turn a creation into acceptance.
Generic existence checks return weak `unknown`, never `accepted`. Unsupported
evaluators return `unknown` with no evidence. Summary counts include unknown,
error, and lifecycle outcomes so all evaluated items reconcile.

`zero_touch` is only true for an accepted PR when comments, submitted reviews,
and commits are available and show no visible non-bot follow-up after the action,
including commits by the PR author.
Missing actor or commit evidence cannot establish zero-touch acceptance.

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
| `create_issue` | closed as completed | dedicated rule |
| `add_comment` | reacted to or replied to | dedicated rule |
| `add_labels` | recorded label addition retained | dedicated rule |
| `add_reviewer` | recorded requested reviewer submitted a review | dedicated rule |
| `update_issue` | intended edit still matches current issue state | dedicated rule |
| `update_pull_request` | intended edit still matches current PR state | dedicated rule |
| `close_issue` | still closed by a visible non-bot actor | dedicated rule |
| `close_pull_request` | still closed unmerged by a visible non-bot actor | dedicated rule |
| `close_discussion` | none yet | no implemented rule yet |
| `create_discussion` | none yet | no implemented rule yet |
| `update_discussion` | none yet | no implemented rule yet |
| `create_pull_request_review_comment` | none yet | no implemented rule yet |
| `submit_pull_request_review` | review affected PR lifecycle | dedicated rule |
| `reply_to_pull_request_review_comment` | none; target existence is unknown | fallback rule |
| `resolve_pull_request_review_thread` | none yet | no implemented rule yet |
| `push_to_pull_request_branch` | recorded pushed commits verified in merged history | dedicated rule |
| `mark_pull_request_as_ready_for_review` | reviewed | dedicated rule |
| `assign_to_agent` | attributable post-assignment agent PR merged | dedicated rule |
| `dispatch_workflow` | recorded dispatched run completed successfully | dedicated rule |
| `autofix_code_scanning_alert` | none; target existence is unknown | fallback rule |
| `create_code_scanning_alert` | none; target existence is unknown | fallback rule |
| `link_sub_issue` | none; target existence is unknown | fallback rule |
| `hide_comment` | none yet | no implemented rule yet |
| `assign_milestone` | recorded milestone identity still assigned | dedicated rule |
| `replace_label` | recorded label replacement delta retained | dedicated rule |
| `update_project` | none; target existence is unknown | fallback rule |
| `update_release` | none; target existence is unknown | fallback rule |
| `noop` | skipped | skipped |
| `report_missing_tool` | skipped | skipped |

## Evaluating Outcomes in Practice

Outcome evaluation does not require an `outcomes:` field in workflow
frontmatter. Configure the actions the workflow may take under
[`safe-outputs:`](/gh-aw/reference/safe-outputs/), then evaluate those actions
after a run:

```bash wrap
printf '%s\n' 1234567890 | gh aw logs --stdin --repo owner/repo --outcomes
```

The command downloads executed safe-output manifests and compact agent-output
fallback evidence, without downloading workflow logs by default. It queries the
current state of each affected GitHub object and includes per-run outcome summaries in
the logs output. Use `--json` for item-level classifications. Use a run ID from
the repository's GitHub Actions page. Specify
`--repo owner/repo` when running outside the target repository.

Evaluation is a snapshot, not a permanent verdict. Run it after the workflow
has produced safe outputs, allow an observation period appropriate to the
output type, and evaluate it again while results remain `pending`. A pull
request may remain pending until review and become accepted when merged; a
comment may need time to receive a reply or reaction.

`gh aw logs --outcomes` is the live outcome-evaluation entry point, including
`gh aw logs episode <run-id-or-url> --outcomes`. Workflow execution
records safe-output evidence; it does not decide whether those outputs were
accepted. `gh aw audit` reports execution evidence without checking live outcome
state. There is no built-in scheduled collector or automatic reevaluation.
Rerun logs with `--outcomes` later for unresolved items; each invocation checks current
GitHub state rather than reusing a cached classification.

Within one collection, identical successful REST and GraphQL reads are shared
across actions and runs. Labels already present in an issue or pull request
response do not require a separate label request. Paginated REST reads request
100 items per page and still fetch all pages. Each action retains its own
timestamp-dependent classification.
This response cache is in memory only: later invocations check GitHub again,
and failed or partial responses are not retained.

Use JSON output to feed a report or inspect the summary:

```bash wrap
gh aw logs weekly-research --outcomes --json |
  jq '.outcomes[] | {run_id, summary}'
```

Repeat this for comparable runs over a stable time window rather than drawing
conclusions from one run. Compare workflows that produce the same kind of safe
output, and examine item-level `object_url`, `outcome_status`, and
`evidence_strength` fields when a summary changes unexpectedly. See
[Measuring Impact](/gh-aw/guides/measuring-impact/) for combining these
results with run volume and AIC.

## Telemetry

Outcome data is derived from safe outputs and later checked against repository state. The system records the safe output produced by the workflow, looks up the affected repository object later, and classifies the observed state into an outcome.

This makes outcome evaluation external and observable. The workflow does not decide whether it succeeded; the repository state does.

Outcome information appears in OpenTelemetry spans and related artifacts. Workflow-level rollups such as accepted counts and acceptance rate are emitted on outcome summary or conclusion spans, and per-item spans can carry more detailed fields such as object type, URL, comments, review activity, and zero-touch acceptance.

For the span-level attribute inventory, see [OpenTelemetry attribute reference](/gh-aw/reference/open-telemetry-attributes/).

To write per-item outcome JSONL for an OTLP reporting pipeline, use:

```bash wrap
gh aw logs weekly-research --outcomes --outcomes-dir ./outcomes
```

For repository-wide or organization-wide reporting, collect these records on a
schedule and aggregate the outcome attributes by workflow, repository, output
type, and observation window.

## Cost and Rollups

Outcomes are most useful when read together with cost data. At the workflow level, the basic questions are how much AIC a workflow spent, how many accepted outcomes it produced, and how much AIC each accepted outcome cost.

The basic dashboard for outcomes is therefore intentionally small: total AIC, total accepted outcomes, AIC per accepted outcome, a trend over time, and a workflow ranking by AIC per accepted outcome.

For simple workflows, a single run is usually the right unit for outcome measurement.

`gh aw logs --outcomes` populates `.outcomes[].summary.cost_per_accepted_outcome` when positive AIC
usage and at least one accepted outcome are available. This is the full
observed run AIC, including detection usage, divided by accepted output items,
not a per-item allocation. Cost is computed from usage artifacts independently
of the agent-only token usage cache.
Unavailable cost and zero accepted outcomes leave this field omitted. If usage
cannot be read or parsed, the command warns and still returns evaluated outcomes
and writes any requested outcome JSONL.

For orchestrated workflows, multiple runs can belong to one logical execution. In that case, the more meaningful unit is the episode. Outcome and cost totals can be rolled up from runs into episodes using simple sums, and then from episodes into workflow totals and repository totals.

### Episode Outcome Summaries

`gh aw logs --outcomes` includes per-run observations in `.outcomes[]` and joins
them with selected episode membership and cost in `.episode_outcomes`.
Use `gh aw logs --episodes --outcomes` to include recorded episode membership
and hops in text output alongside these summaries. The combined flags perform
the episode-outcome join in memory, without a second command, JSONL export,
or additional GitHub requests for the join itself. Standard logs JSON membership
and outcome summaries are preserved without the explicit `--episodes` flag.
Manifests are collected alongside usage artifacts; full agent artifacts and
workflow logs are not required. Episode membership remains bounded to selected
runs. `gh aw logs episode <seed> --outcomes` also checks outcomes live when
membership comes from a cached snapshot; its retrieval API cap covers artifacts
and discovery, not the additional live outcome queries.

`gh aw logs outcomes-episodes` joins previously recorded outcome observations to the episode
membership from a logs report. It is an offline report, not another evaluator:

```bash wrap
gh aw logs --outcomes --outcomes-dir ./outcomes --verbose --json > logs.json
gh aw logs outcomes-episodes --logs logs.json --outcomes-dir ./outcomes --json
```

Record outcomes for each selected source run. Use `--output <directory>` on
both commands when downloaded `run-*` artifacts are outside the default
`.github/aw/logs` directory. The episode command reads usage artifacts directly,
including detection usage, rather than summing agent-only cached cost.
Accounting sources are selected separately for each execution phase before
summing independent stream totals. Agent JSON remains a fallback when agent
JSONL has no usable cost; an executed phase without accounting qualifies cost
coverage as partial.
Both compact and verbose logs JSON retain all canonical episodes, including standalone episodes.
A successfully evaluated empty or all-skipped manifest establishes outcome coverage
for that selected run. A missing manifest does not establish zero outputs and
keeps outcome coverage partial.

Membership can also come from [`gh aw logs episode`](/gh-aw/setup/cli/#logs-episode-run-id-or-url).
Its bounded retrieval report uses the same episode contract. The outcome join
reads each run's recorded `logs_path` beneath `--output`, including isolated
repository/attempt directories; conflicting paths or paths outside that root
are errors. Collection limits remain membership gaps and suppress efficiency
claims; this offline join does not perform additional discovery.

The command reads all `.jsonl` files directly inside `--outcomes-dir`. New
per-item records include `item_id` (a SHA-256 digest of the source action record,
with an occurrence number for indistinguishable records) and
`run_item_count` (the number of evaluated actions). The join matches `run_id`
against episode `run_ids`, selects the latest `checked_at` for each source-run
item, and reuses CLI outcome classifications. Repeated snapshots do not add
accepted outcomes, and a later rejection replaces a previous acceptance.
Conflicting evaluation fields at the latest observation time, including
evidence strength, objective value, and item count, produce an error rather
than an arbitrary verdict. A newer observation supersedes older conflicts.
Reordering or reserializing unchanged source records does not change action
identities; distinct actions against the same object remain distinct items.

JSON returns `episodes[]` and `unmatched_outcome_run_ids`. Each episode contains
the selected `run_ids`, latest `items`, the existing outcome `summary`, and:

| Field | Meaning |
| --- | --- |
| `membership_scope` | Always `selected_runs`; the logs selection does not prove full-episode membership. |
| `membership_coverage` | Optional lineage/collection scope, status, reasons, and truncation evidence copied from the logs, separate from outcome/cost coverage. |
| `outcome_coverage` | `complete` or `partial` for selected runs only, based on available observations, item identity, and recorded item counts. |
| `outcome_run_ids`, `missing_outcome_run_ids` | Selected runs with and without recorded outcomes. No observations do not prove that a run produced no outputs. |
| `observed_aic` | Sum of available selected-run usage; `null` when unavailable. An explicit recorded zero remains `0`, not unknown. |
| `cost_coverage` | `available`, `partial`, or `unavailable` for selected runs. Usage warnings qualify cost as partial. |
| `cost_run_ids`, `missing_cost_run_ids` | Selected runs with and without usable AIC evidence. |
| `missing_run_ids` | Episode member IDs absent from the supplied run records. |
| `cost_per_accepted_outcome` | Observed AIC divided by accepted items, or `null` when there are no accepted items, incomplete membership/evidence/cost, or ambiguous legacy identities. A valid zero cost yields `0`. |
| `qualifications` | Scope and evidence limitations, including unresolved outcomes. |

Pending, unknown, and error outcomes remain unresolved rather than becoming
accepted. A ratio with unresolved items is an observed snapshot, not a final
efficiency verdict. Partial AIC is never divided by accepted items to imply a
complete cost result. Outcomes whose source runs are outside selected
membership are listed separately and excluded from all episode counts.

Older JSONL without `item_id` is conservatively coalesced by source run, action
type, repository, target, and creation time only when no precise item identities
exist for that group. Precise observations supersede matching legacy rows once
each item's latest observation is at least as recent as the latest legacy row.
A newer legacy row produces an error requiring reevaluation rather than
overriding ambiguous precise items or adding an extra item.
Indistinguishable legacy-only actions can be undercounted, so these reports are
qualified and efficiency is unavailable; reevaluate those runs to record action
identities and counts. Earlier positional IDs (`item-1`, `item-2`, and so on)
also lack reorder-stable identity and disable efficiency when retained.
Content-derived identities supersede matching positional observations under
the same timestamp rule, rather than counting them again. The report does
not infer missing episode members, reevaluate GitHub objects, or automatically
schedule observations. Generate its schema with
`gh aw json-schema outcomes-episodes`.

The outcomes model is deliberately narrow. It does not try to estimate the full business value of a workflow, replace human judgment for nuanced quality questions, combine deterministic compute cost and inference cost into one synthetic score, or solve overlap and duplicate-work analysis in the first version.

Those questions may matter later, but they are separate from the base outcomes model described here.

## Learn More

- [Cost Management](/gh-aw/reference/cost-management/) explains how workflow cost is measured and reduced.
- [OpenTelemetry attribute reference](/gh-aw/reference/open-telemetry-attributes/) describes the span attributes and artifacts that carry workflow telemetry.
- [Safe Outputs](/gh-aw/reference/safe-outputs/) explains how workflows produce constrained actions.
- [Safe Output Outcome Evaluation Specification](https://github.com/github/gh-aw/blob/main/specs/safe-output-outcome-evaluation.md) defines the detailed evaluation logic for each safe output type.
