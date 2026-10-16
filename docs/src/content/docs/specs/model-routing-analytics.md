---
title: Model Routing Analytics Design
description: Draft design for a model routing analytics page built only from unified session files, with metric definitions, source events, and proposed records for missing data.
sidebar:
  order: 1366
version: "0.1.0"
status: Draft
publication_date: "2026-10-16"
editors:
  - name: GitHub Agentic Workflows Team
    organization: GitHub
---

# Model Routing Analytics Design

**Version**: 0.1.0<br>
**Status**: Draft<br>
**Publication Date**: 2026-10-16<br>
**Editors**: GitHub Agentic Workflows Team (GitHub)<br>
**This Version**: [model-routing-analytics](/gh-aw/specs/model-routing-analytics/)<br>
**Latest Version**: This document

---

## Abstract

This document designs a "Model routing" analytics page that reads only ingested unified session files (`usage/aw_session.jsonl`, one per run). It defines the page's views, the exact definition of every metric, and the session event and field each metric reads. It confirms each known data gap against the [Unified Agent Session Specification](/gh-aw/specs/unified-agent-session-specification/) and the current collector, and proposes a runner- or proxy-written record for each gap. Nothing in this document is implemented yet.

## Status of This Document

This is a design proposal for review. It does not change the session collector, AWF, the router, or any specification. Section 6 lists the record changes that follow once the design is agreed; each becomes a change to the unified session specification with its own requirement IDs. Statements about current behavior describe unified session specification 1.10.0, the gh-aw collector at publication, and AWF `model-routing/v0.28.51` records.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Input Contract](#3-input-contract)
4. [Run Model](#4-run-model)
5. [Views and Metrics](#5-views-and-metrics)
6. [Data Gaps and Proposed Records](#6-data-gaps-and-proposed-records)
7. [Rollout](#7-rollout)
8. [Open Questions](#8-open-questions)
9. [References](#9-references)
10. [Change Log](#10-change-log)

---

## 1. Introduction

### 1.1 Purpose

A workflow opts into routing with `engine.model-routing` (`goal`, `mode`, and optional `allowed-models`); `models.allowed` and `models.blocked` remain the request policy. At run start AWF's router classifies the task with a small LLM call, ranks the live candidate models and efforts from its routing tables, and AWF records the selection. The main agent runs on the selected model and effort, sub-agents can run on other models, and AWF's api-proxy records every request with its AI-credit (AIC) cost. A routing failure fails the run with exit code 78.

The page answers, per workflow, repository, engine, and time range:

1. **Adoption**: routed versus unrouted runs by goal, mode, and engine.
2. **Decisions**: which model and effort the router picked, how stable the pick is, and the classifier's labels.
3. **Cost**: AIC per run split into main agent, sub-agents, classifier, and threat detection, compared with unrouted runs.
4. **Deviations**: requests and credits that did not use the router's pick, and why.
5. **Failures**: routing failures, unavailable models, budget exhaustion, and steering notices.
6. **Outcomes**: run and job results and safe-output results by selected model and effort.
7. **Drill-down**: one run's routing timeline, linked to the Actions run.

### 1.2 Scope

In scope: views, metric definitions, reading rules for current and older session files, and proposed records for missing data. Out of scope: the page implementation and hosting, collector or AWF changes (Section 6 proposes them), and router ranking quality.

### 1.3 Design goals

- Read only unified session files. No engine-native files, AWF logs, `aw_info.json`, or GitHub API calls.
- Attribute models and credits from runner- or proxy-written evidence. Agent-writable records are shown as advisory and never decide attribution alone.
- Treat missing fields as unknown. Never turn absence into zero, success, or "agent".
- Load every session file the collector has produced, including files written before newer events existed.
- Show only metadata the session specification already keeps: no prompts, agent text, tool arguments, error bodies, or free-form diagnostics.

### 1.4 Terminology

| Term | Meaning |
| --- | --- |
| Run | One accepted session file. Its identity is defined in [Section 4.1](#41-run-identity-and-time). |
| Routed run | A run whose routing state ([Section 4.3](#43-routing-state)) is `selected`, `selection_failed`, or `harness_reported`. |
| Pick | The router's selected model and effort for the main agent. |
| Request record | A `firewall.model_routing` event with `stage: "request"`. AWF writes one for each routed request. |
| Accounting record | A `firewall.token_usage` event, or a `usage.report` event when [Section 4.4](#44-request-attribution) selects it. |
| Proxy-written | Written by AWF's api-proxy or router outside the agent sandbox (`firewall.*`, `model.inventory`). |
| Runner-written | Written by gh-aw job steps outside the agent process (`workflow.info`, `workflow.job_result`, `agent.execution`, `safe_output.result`, `safe_output.error`, `detection.result`, `session.collection`). |
| Engine-reported | Written by the engine process (`subagent.*`, `session.result`). The unified session specification accepts these as served-model evidence for sub-agents (T-UAS-073). |
| Advisory | Agent-writable and never authoritative: `model_routing.outcome` (from `agent/awf-routing-outcome.json`) and `safe_output.request`. |
| Unknown | A value that no accepted event supplies. Rendered as "unknown" with the number of runs it affects. |

## 2. Conformance

The key words MUST, MUST NOT, SHOULD, and MAY are to be interpreted as described in RFC 2119. Requirements labelled `MRA-nnn` apply to the page's reader. Proposed records in Section 6 are not requirements until they are added to the unified session specification.

## 3. Input Contract

### 3.1 Session file acceptance

**MRA-001 — Format header.** The reader MUST read the first record. When it is a `session.format` header with `data.version` equal to `1`, the file is accepted. A file with a higher version MUST be counted as "unsupported format" and not read. A file without a header MUST be read as a legacy file and counted as such in the coverage panel.

**MRA-002 — Coverage.** The reader MUST retain the file's `session.collection` event (`sources`, `absentComponents`, `warnings`, `untimedEvents`) and use it to distinguish a missing source from a present empty source, as the unified session specification requires (T-UAS-063). Collection warnings, such as `invalid_model_inventory`, MUST be counted per run.

**MRA-003 — Unknown event types.** The reader MUST ignore event types and fields it does not recognize, so that additive records from Section 6 do not break older readers.

### 3.2 Evidence trust

| Evidence | Writer | Used for | Never used for |
| --- | --- | --- | --- |
| `firewall.model_routing`, `firewall.token_usage`, `firewall.steering`, `firewall.event`, `model.inventory` | AWF api-proxy and router | Routing state, pick, labels, per-request models, tokens, credits, deviations, failures | — |
| `workflow.info` | gh-aw runner (`aw_info.json`) | Run identity, versions, engine, observed model, fallback routing state | Overriding a proxy selection or failure record ([Section 4.3](#43-routing-state)) |
| `workflow.job_result`, `agent.execution`, `execution.result`, `detection.result`, `safe_output.result`, `safe_output.error`, `grader.result` | gh-aw runner | Outcomes and failure categories | — |
| `subagent.*`, `session.result.agentMetrics` | Engine | Sub-agent definition names, instance lifecycle, configured effort | Credits or served models without a proxy match ([Section 4.4](#44-request-attribution)) |
| `model_routing.outcome`, `safe_output.request` | Agent-writable | Advisory comparison columns, "requested" counts | Routing state, pick, attribution, "applied" counts |

**MRA-010 — Proxy precedence.** Routing state, pick, and per-request attribution MUST come from proxy-written events when any exist in the run. `workflow.info.modelRouting` MAY fill a value only when no proxy record supplies it. `model_routing.outcome` MUST NOT fill any value.

`workflow.info.modelRouting.status` is not purely proxy-derived. The harness compares the final proxy record with the advisory `awf-routing-outcome.json`, and the golden fixture `copilot-budget-exhausted-steering` records `status: "rejected"` with `failureCode: "harness_selection_mismatch"` for a run whose proxy records show a selection followed by an `ai_credits_limit_exceeded` failure. MRA-010 keeps the page correct for such runs.

### 3.3 Field names and older files

Current collectors project essential fields in camelCase. Older session files can retain AWF's raw snake_case record, including fields that the current projection drops (`ranked_choices`, `eligible_choices`, `catalogue_overlap`, `unavailable`, `upstream_endpoint`, request `status`). The golden fixture `codex-routed-provider-unavailable` is such a file.

**MRA-020 — Field aliases.** For every field in this document, the reader MUST accept both spellings: the camelCase name, then the snake_case source name used by the collector's projection table in `actions/setup/js/unified_session_payload.cjs` (for example `selectedModel` / `selected_model`, `schema` / `_schema`, `requestId` / `request_id`, `aic` / `ai_credits_this_response`). Fields present only in older raw records MAY be used when present and are unknown otherwise.

**MRA-021 — Legacy endpoint deviations.** A request record with `routed: "deviated"` and `deviations` exactly `["endpoint"]` MUST be read as `as_selected` when the run's selection `schema` is below `model-routing/v0.28.39`, matching `normalizeLegacyEndpointDeviation` in `pkg/cli/model_routing.go`. The run MUST be flagged "legacy endpoint normalization".

**MRA-022 — Model identifiers.** For grouping, a model identifier is compared after removing any provider prefix up to and including the last `/` (`github-copilot/gpt-5.6-sol` becomes `gpt-5.6-sol`). Dated served IDs (`claude-haiku-4-5-20251001`) are kept as observed; the reader MUST NOT map them to undated names without an alias table in the session.

### 3.4 Unknown values

**MRA-030 — Absence is unknown.** A missing field or event MUST be rendered as unknown, never as zero, success, or a default category. Specifically:

- A missing `purpose` on an accounting record is unknown, not agent traffic (T-UAS-070, "Firewall usage purpose").
- A missing `steering` field means "no notice recorded", not "no notice delivered" (T-UAS-071).
- A missing `aic` makes that request's cost unknown. The reader MUST NOT reprice it.
- A missing safe-output result or grader result is unknown, not success (T-UAS-062). The page applies the same rule to a missing job result.

**MRA-031 — Partial sums.** Every summed metric MUST carry a coverage count: the number of contributing items with a known value and the number with an unknown value. A per-run sum with any unknown item is "partial" and MUST be excluded from medians, percentiles, and comparisons, but MAY be shown as a lower bound in the drill-down.

**MRA-032 — Ratios.** A ratio's denominator MUST contain only runs or requests for which the numerator's condition is decidable. The excluded count MUST be displayed alongside the ratio.

**MRA-033 — No overlapping sums.** The reader MUST NOT add overlapping observations (T-UAS-062): `firewall.token_usage` and `usage.report` for the same phase, `totalAic` checkpoints, and per-agent snapshots in `session.result.agentMetrics` are alternatives, not addends.

### 3.5 Privacy

**MRA-040 — Metadata only.** The page MUST NOT display or store, even transiently in derived tables: prompt events, `user.message`, `assistant.*`, tool inputs and results, `session.error`, any `message`, `detail`, `error`, or `reason` text field (including `firewall.steering.message` and `reason`, `safe_output.*.message` and `error`, and `subagent.failed.error`), the raw AWF fields `conversation_sha256`, `interaction_id`, and `response_body` when present in older raw records, and `model_endpoint.mismatch.detail` and `fix` outside the single-run drill-down. Identifier-shaped values (`[A-Za-z0-9_.:/-]{1,128}`) such as failure codes, error codes, labels, and model IDs MAY be displayed.

`degradedReason` MUST be displayed only when it is identifier-shaped. Classifier labels that the source redacted (`"[redacted]"`) MUST be shown as "redacted", distinct from the router's own `unknown` label.

## 4. Run Model

### 4.1 Run identity and time

| Attribute | Definition | Source |
| --- | --- | --- |
| Repository | `repository` | `workflow.info` |
| Workflow | `workflow` (display name) | `workflow.info` |
| Run ID | `runId` | `workflow.info` |
| Actions link | `https://github.com/{repository}/actions/runs/{runId}` | `workflow.info` |
| Run time | Minimum `provenance.timestampMs` over all events in the file | Any timestamped event |

**MRA-050 — Identity.** A file without `workflow.info`, or without `repository` and `runId`, MUST be counted as "unidentified" and excluded from per-workflow views. When two files share `repository` and `runId` (re-run attempts), the reader MUST keep the file with the later run time and count the duplicate; [G-12](#g-12-run-attempt-workflow-file-and-server) proposes the fields that make this exact.

**MRA-051 — Time.** A run with no timestamped event has unknown time; it MUST be excluded from time-filtered views and counted as "untimed". Time buckets use UTC days or ISO weeks.

### 4.2 Dimensions

| Dimension | Source | Notes |
| --- | --- | --- |
| Engine | `workflow.info.engineId` (`copilot`, `claude`, `codex`, `pi`, others) | Copilot CLI and Copilot SDK are indistinguishable today; see [G-9](#g-9-engine-harness). |
| gh-aw version | `workflow.info.cliVersion` | Absent for unreleased builds. |
| AWF version | `workflow.info.awfVersion` | |
| Router version | Selection record `router.version`, else `workflow.info.modelRouting.routerVersion` | Router package version (for example `0.1.3`), not the AWF release; see [G-3](#g-3-router-routing-table-and-classifier-identity). |
| Goal | Selection record `objective.goal` (`cost`, `cost-speed`) | Configured value. Unknown for runs without a selection record until [G-1](#g-1-model-configuration). |
| Configured mode | Selection record `objective.mode` (`auto`, `economy`, `balanced`, `robust`) | |
| Resolved mode | Selection record `mode`, else `workflow.info.modelRouting.mode` | `auto` resolves to `economy`, `balanced`, or `robust`. |
| Pick | [Section 5.2](#52-decisions) | |
| Labels | Selection record `labels.task_type`, `labels.scope`, `labels.task_complexity` | |

### 4.3 Routing state

**MRA-060 — Routing state.** Each run has exactly one routing state, decided in this order using agent-phase events only:

| Order | Condition | State |
| ---: | --- | --- |
| 1 | `model.config.routing.enabled` is `false` ([G-1](#g-1-model-configuration)) | `unrouted` |
| 2 | A `firewall.model_routing` record with `stage: "selection"` exists | `selected` |
| 3 | A `firewall.model_routing` record with `stage: "failure"` exists and no selection record exists | `selection_failed` |
| 4 | `workflow.info.modelRouting.status` is present | `harness_reported` (status shown as reported: `selected`, `failed`, `rejected`, `pending`, or `unavailable`) |
| 5 | Agent-phase `firewall.token_usage` exists and no `firewall.model_routing` record exists | `unrouted_inferred` |
| 6 | Otherwise | `unknown` |

`unrouted_inferred` relies on AWF writing its classification record before any agent request when routing is enabled. Views MUST show `unrouted` and `unrouted_inferred` separately until [G-1](#g-1-model-configuration) makes the inference unnecessary. When several selection records exist, the last by timestamp is the run's selection and the earlier count is shown as re-selections.

A failure record that follows the selection, such as `ai_credits_limit_exceeded` during execution, does not change the state from `selected`; it is a failure after selection ([Section 5.5](#55-failures)).

### 4.4 Request attribution

**MRA-070 — Accounting source.** For each phase (`agent`, `detection`, `evals`), the accounting records are that phase's `firewall.token_usage` events when any exist, otherwise that phase's `usage.report` events. The two MUST NOT be combined. Per-request cost is `aic`. `totalAic` is a cumulative checkpoint per AWF process and MUST NOT be summed; it MAY be used to cross-check completeness.

Threat detection is identified by provenance phase `detection` (sources under `threat-detection/`), not by `purpose`. AWF writes no detection purpose; in every golden fixture under `pkg/cli/testdata/model_routing_golden/` the only recorded purpose is `routing_classification`. Detection is not routed and uses its configured detection model.

**MRA-071 — Buckets.** Each accounting record belongs to exactly one bucket, assigned in this order:

| Order | Bucket | Rule |
| ---: | --- | --- |
| 1 | Detection | `provenance.phase` is `detection`. |
| 2 | Evals | `provenance.phase` is `evals`. Shown separately; excluded from routing cost. |
| 3 | Classifier | `purpose` is `routing_classification`. |
| 4 | Sub-agent `{agentName}`, exact | The record's `requestId` equals a `subagent.request.requestId`. |
| 5 | Sub-agent `{agentName}`, correlated | No exact match, and the record matches one `subagent.request` by `model`, the token quartet (`inputTokens`, `outputTokens`, `cacheReadInputTokens`/`cacheReadTokens`, `cacheCreationInputTokens`/`cacheWriteTokens`), and order, as the audit correlator does today. Each `subagent.request` matches at most one record. |
| 6 | Main agent | `purpose` is `agent`, or the record's `requestId` matches a request record and the run has no `subagent.*` events. |
| 7 | Agent, unsplit | The record's `requestId` matches a request record and the run has `subagent.*` events, or `purpose` is `subagent` without a match. |
| 8 | Unattributed | Every other agent-phase record, including all records with no `purpose` and no request record. |

Sub-agent rows are keyed by `agentName` (T-UAS-073); when `agentName` is absent, the row uses `agentDisplayName`. The served model of a sub-agent row is the accounting record's `model`, never `model_routing.outcome`. The engine snapshot `session.result.agentMetrics[*].totalNanoAiu / 10^9` MAY be shown in a separate "engine-reported" column for sub-agents with no matched record; it is never added to proxy totals.

**MRA-072 — Request join.** An accounting record joins its request record by `requestId`. The join supplies `routed`, `deviations`, `requestedModel`, `requestedEffort`, `selectedModel`, `selectedEffort`, and `outcome`. A request record without an accounting record (for example a request rejected with HTTP 403) has unknown cost.

### 4.5 Deviation causes

**MRA-080 — Cause.** Each request record with `routed: "deviated"` (after MRA-021) gets the first matching cause:

| Order | Cause | Rule |
| ---: | --- | --- |
| 1 | Sub-agent model fallback | The request is in a sub-agent bucket and the instance has `subagent.model_unavailable` (`declaredModel` → `model`), `subagent.failed.errorCode` `SUBAGENT_MODEL_UNAVAILABLE`, or `subagent.completed.firstDispatchedModel` different from `model`. |
| 2 | Sub-agent model | The request is in a sub-agent bucket. Split by `subagent.started.modelSelectionSource` (for example `agent_definition_default`, `custom-agent`). Expected: routing chooses only the main agent's model. |
| 3 | Harness substitution | `workflow.info.modelSelection.status` is `substituted` and `requestedModel` equals `modelSelection.effectiveModel`. |
| 4 | Effort change | `deviations` is exactly `["effort"]`. Shown as `requestedEffort` → `selectedEffort`. |
| 5 | Endpoint change | `deviations` is exactly `["endpoint"]`. |
| 6 | Unexplained | Any other deviated request, including those in the "agent, unsplit" and "unattributed" buckets. |

Routing cannot be combined with `engine.fallback-models`, and a routing failure fails the run instead of falling back to a fixed model, so main-agent fallbacks appear only as harness substitution (cause 3) or as unexplained deviations.

## 5. Views and Metrics

Every view accepts the filters workflow, repository, engine, routing state, goal, mode, AWF version, and time range. Comparisons use a configurable minimum sample, `N_min` (default 10 runs per group); a group below `N_min` is shown as "insufficient data" rather than as a value.

### 5.1 Adoption

| Metric | Definition | Source |
| --- | --- | --- |
| Runs | Accepted, identified runs in the filter. | `session.format`, `workflow.info` |
| Routed runs | Runs whose state is `selected`, `selection_failed`, or `harness_reported`. | [Section 4.3](#43-routing-state) |
| Unrouted runs | Runs whose state is `unrouted`, with `unrouted_inferred` as a separate series. | [Section 4.3](#43-routing-state) |
| Unknown-state runs | Runs whose state is `unknown`. | [Section 4.3](#43-routing-state) |
| Routed share | Routed runs ÷ (routed + unrouted + unrouted_inferred runs). Unknown-state runs are excluded and shown as coverage. | [Section 4.3](#43-routing-state) |
| Routed runs by goal, configured mode, resolved mode | Count of `selected` runs grouped by `objective.goal`, `objective.mode`, and `mode`. | Selection record |
| Routed runs by engine | Routed runs grouped by `engineId`. | `workflow.info` |
| Degraded classification share | Selections with `degradedClassification: true` ÷ selections where the field is present. | Selection record |
| Classifier attempts | Distribution of `classifierAttempts`. | Selection record |
| Routed workflows | Distinct (repository, workflow) with at least one routed run. | `workflow.info` |

### 5.2 Decisions

| Metric | Definition | Source |
| --- | --- | --- |
| Pick | (model, effort) where model is the selection record's `selectedModel` normalized by MRA-022 (else `wireModel`), and effort is `selectedEffort`. For `harness_reported` runs: `workflow.info.modelRouting.model` or `wireModel`, and `effort`. | Selection record, `workflow.info.modelRouting` |
| Applied effort | `workflow.info.modelRouting.appliedEffort` when it differs from the pick's effort. | `workflow.info.modelRouting` |
| Pick share | Runs with pick *p* ÷ runs with a known pick. | Derived |
| Distinct picks | Number of distinct picks per workflow in the range. | Derived |
| Modal pick share | max over *p* of runs with pick *p* ÷ runs with a known pick, per workflow. Higher means more stable. | Derived |
| Pick change rate | Among a workflow's runs with a known pick, ordered by run time, the fraction of adjacent pairs whose picks differ. Requires at least two runs. | Derived |
| Label-conditioned change rate | Pick change rate computed within each label cell (`task_type`, `scope`, `task_complexity`) and averaged, weighted by pairs. Separates task variation from router variation. | Selection record `labels` |
| Change context | For each pick change within a label cell, whether `awfVersion`, router version, or the agent-phase `model.inventory` model set also changed between the two runs. | `workflow.info`, selection `router`, `model.inventory` |
| Label distribution | Count of runs per `task_type`, `scope`, and `task_complexity`. | Selection record `labels` |
| Label-to-pick matrix | Run counts by label cell and pick. | Selection record |
| Classifier model | Distribution of `classifierModel` and `classifierEffort`. | Classification or selection record |
| Router latency | p50 and p95 of selection `latencyMs`. | Selection record |
| Candidate pool | `eligibleChoices`, `catalogueOverlap`, and the rank of the pick within `rankedChoices`. Available only in older raw files until [G-2](#g-2-router-candidate-pool-and-predictions). | Selection record |

### 5.3 Cost

All costs are AIC from `aic` on accounting records ([Section 4.4](#44-request-attribution)). A run's cost is "complete" when every accounting record in the relevant buckets has `aic`.

| Metric | Definition | Source |
| --- | --- | --- |
| Agent-phase AIC | Sum of `aic` over buckets 3–8 (classifier, sub-agent, main agent, unsplit, unattributed). | `firewall.token_usage` |
| Classifier AIC | Sum of `aic` in the classifier bucket. | `firewall.token_usage.purpose` |
| Main-agent AIC | Sum of `aic` in the main-agent bucket. | [Section 4.4](#44-request-attribution) |
| Sub-agent AIC by definition | Sum of `aic` per `agentName`, with exact and correlated amounts shown separately. | `firewall.token_usage`, `subagent.request` |
| Unsplit and unattributed AIC | Sum of `aic` in buckets 7 and 8. Shown, never redistributed. | `firewall.token_usage` |
| Detection AIC | Sum of `aic` over the detection phase's accounting records. | `firewall.token_usage` or `usage.report`, phase `detection` |
| Run AIC | Agent-phase AIC + detection AIC. Complete only when both parts are complete. | Derived |
| Classifier share | Classifier AIC ÷ agent-phase AIC. | Derived |
| Tokens | The same splits over `usage.inputTokens`, `usage.outputTokens`, `usage.reasoningOutputTokens`, `usage.cacheReadInputTokens`, and `usage.cacheCreationInputTokens`. | `firewall.token_usage.usage` |
| AIC by served model | Sum of `aic` grouped by the accounting record's `model`. | `firewall.token_usage.model` |
| AIC by pick | Median and p90 of complete agent-phase AIC per pick. | Derived |
| Routed versus unrouted | For one workflow and time range: median and p75 of complete agent-phase AIC for `selected` runs, and the same for `unrouted` and `unrouted_inferred` runs, labelled with the unrouted runs' `workflow.info.model`. Delta = (routed median − unrouted median) ÷ unrouted median. Requires `N_min` complete runs in each group. | Derived |
| Baseline estimate | Per run, the repriced cost on the configured default model. Available only after [G-6](#g-6-baseline-price). | `model.baseline_estimate` |
| Cost trend | Weekly median of complete agent-phase AIC, one series per routing state, and one per pick for the top five picks. | Derived |
| Budget use | Run AIC ÷ `model.config.maxAiCredits`. Until [G-1](#g-1-model-configuration), the highest AIC steering threshold reached ([Section 5.5](#55-failures)) is shown instead. | `model.config`, steering |

When runs carry an `experiment.assignment` event whose variants differ in routing, the routed-versus-unrouted comparison SHOULD be computed within the experiment's variants and labelled as such. Otherwise the comparison is observational: routed and unrouted runs of a workflow typically come from different periods and tasks, and the page MUST say so next to the delta.

### 5.4 Deviations

| Metric | Definition | Source |
| --- | --- | --- |
| Deviated request share | Request records with `routed: "deviated"` ÷ request records with `routed` present, after MRA-021. | `firewall.model_routing` (`stage: "request"`) |
| Deviated credit share | Sum of `aic` of accounting records joined to deviated request records ÷ sum of `aic` of accounting records joined to any request record. Joined records without `aic` are excluded and counted. | MRA-072 |
| Deviations by axis | Count of deviated requests per value in `deviations` (for example `model`, `effort`, `endpoint`). A request counts once per value. | Request record `deviations` |
| Deviations by cause | Count and AIC per cause. | [Section 4.5](#45-deviation-causes) |
| Requested versus selected | Matrix of (`requestedModel`, `requestedEffort`) against (`selectedModel`, `selectedEffort`) for deviated requests. | Request record |
| Provider deviation | Request records whose `provider` differs from the selection's `selectedProvider`. | Request and selection records |
| Observed model differs | Runs where `workflow.info.model` differs from `workflow.info.modelRouting.wireModel` after MRA-022. | `workflow.info` |
| Harness substitution | Runs with `workflow.info.modelSelection.status: "substituted"`, grouped by `configuredModel`, `resolvedModel`, and `effectiveModel`. Copilot only. | `workflow.info.modelSelection` |
| Model/endpoint mismatch | Runs with a `model_endpoint.mismatch` event, grouped by `phase`, `configured_model`, `resolved_model`, and `wire_api`. | `model_endpoint.mismatch` |
| Alias resolution | For each (workflow, `role`, `configured_model`), the sequence of `resolved_model` over run time. An "alias move" is a run whose `resolved_model` differs from the previous run's. Failed resolutions grouped by `cause`. | `model.resolution` (proposed in pull request #69487; unknown for files without it) |
| Advisory disagreement | Runs where `model_routing.outcome.wireModel` or `effort` differs from the proxy pick. Informational only. | `model_routing.outcome` |

### 5.5 Failures

| Metric | Definition | Source |
| --- | --- | --- |
| Selection failures | Runs in state `selection_failed`, grouped by the failure record's `code`. | `firewall.model_routing` (`stage: "failure"`) |
| Failures after selection | `selected` runs with a later failure record, grouped by `code` (for example `ai_credits_limit_exceeded`, `provider_unavailable`). | `firewall.model_routing` |
| Harness-reported failures | `harness_reported` runs with `modelRouting.status` other than `selected`, grouped by `failureCode`. | `workflow.info.modelRouting` |
| Exit 78 runs | Runs with `agent.execution.exitCode` equal to 78, cross-tabulated with failure `code` and `agent.execution.categories`. Exit 78 with no proxy failure record is flagged "cause not recorded". | `agent.execution` |
| Budget exhaustion | Runs with failure `code` `ai_credits_limit_exceeded` or with `max_ai_credits_exceeded` in `agent.execution.categories`. | `firewall.model_routing`, `agent.execution` |
| Steering reach | Per run and steering type (`ai_credit`, `timeout`, `token`), the highest `threshold` in either `firewall.token_usage.steering` or `firewall.steering`. The share of runs reaching 80, 90, 95, and 99 uses only runs with at least one steering field or event in the denominator; other runs are "no notice recorded". | `firewall.token_usage.steering`, `firewall.steering.threshold` |
| Rejected requests | Request records with `outcome: "rejected"`, grouped by `requestedModel`; runs with `model_policy_violation` or `awf_api_proxy_blocking_requests` in `agent.execution.categories`. | Request record, `agent.execution` |
| Failed and aborted requests | Request records with `outcome` `failed` or `aborted`, grouped by `requestedModel` and `requestedEffort`. | Request record |
| Upstream errors | `firewall.event` events with `event: "UPSTREAM_ERROR_RESPONSE"`, grouped by `status` and, through `requestId` (MRA-072), by requested model and effort. Error code available only after [G-5](#g-5-upstream-errors). | `firewall.event` |
| Unavailable models | Count of `subagent.model_unavailable` events (`declaredModel` → `model`), `subagent.failed` events with `errorCode` `SUBAGENT_MODEL_UNAVAILABLE`, runs with `model_not_supported_error` in `agent.execution.categories`, and picks absent from the run's agent-phase `model.inventory` when that inventory has `modelsFetchComplete: true`. | `subagent.*`, `agent.execution`, `model.inventory` |
| Failure clustering | For each pick, failing runs ÷ runs with that pick, where failing means any of the rows above. For request-level failures, the same ratio per (`requestedModel`, `requestedEffort`). | Derived |

### 5.6 Outcomes

| Metric | Definition | Source |
| --- | --- | --- |
| Agent job result | `result` of the `workflow.job_result` event with `jobId: "agent"` (`success`, `failure`, `cancelled`, `skipped`). | `workflow.job_result` |
| Dependency outcome | `failure` if any job result is `failure`; else `cancelled` if any is `cancelled`; else `success` if every non-skipped job result is `success`; unknown when the run has no job results. The conclusion job's own result is not in the snapshot. | `workflow.job_result` |
| Agent exit | `agent.execution.exitCode`, and `execution.result.conclusion` for the agent phase. | `agent.execution`, `execution.result` |
| Detection verdict | `detection.result.conclusion`, with `promptInjection`, `secretLeak`, and `maliciousPatch` flags. | `detection.result` |
| Safe outputs requested | Count of `safe_output.request` events by `type`. Advisory: requests are not results. | `safe_output.request` |
| Safe outputs applied | Count of `safe_output.result` events by `type` and `status`. Unknown when `session.collection` lists no `safe-output-items.jsonl` source; zero when it lists the source with no events. | `safe_output.result`, `session.collection` |
| Safe-output errors | Count of `safe_output.error` events by `type` and `errorCode`. | `safe_output.error` |
| Apply rate | Applied ÷ requested per run, for runs where both sources are present. | Derived |
| Operational value | `value` of the `grader.result` with `id: "operational-value"`, with its `unit` and `direction`. | `grader.result` |
| Delayed outcome | Latest `result` per safe-output item (`accepted`, `rejected`, `pending`, and others). Available only after [G-4](#g-4-run-outcome). | `safe_output.outcome` |
| Success rate by pick | Runs with agent job result `success` ÷ runs with a known agent job result, per pick, with a 95% Wilson interval. Also per label cell. | Derived |
| Cost per success | Sum of complete agent-phase AIC ÷ runs with agent job result `success`, per pick. | Derived |

The page answers "do cheaper picks hurt results" by plotting success rate and cost per success per pick within one label cell, so that harder tasks routed to stronger models do not bias the comparison.

### 5.7 Run drill-down

The drill-down shows one run, with its Actions link ([Section 4.1](#41-run-identity-and-time)), versions, engine, routing state, and coverage panel.

**Timeline.** Timestamped events, ordered by `provenance.timestampMs` and then file order:

| Row | Source | Columns |
| --- | --- | --- |
| Classification attempt | `firewall.model_routing` (`stage: "classification"`) | `attempt`, `classifierModel`, `classifierEffort`, classifier AIC from the classifier bucket |
| Selection | `firewall.model_routing` (`stage: "selection"`) | Objective, resolved mode, labels, pick, `selectedId`, `endpoint`, router, `latencyMs`, degraded state, ranked choices when present |
| Request | Request record joined with its accounting record | Bucket, requested and selected model and effort, `routed`, `deviations`, `outcome`, tokens, `aic`, `durationMs`, steering notice |
| Unmatched accounting | Accounting record without a request record | Bucket, model, tokens, `aic` |
| Sub-agent lifecycle | `subagent.started`, `subagent.completed`, `subagent.failed`, `subagent.model_unavailable` | `agentName`, `agentDisplayName`, `model`, `reasoningEffort` from `subagent.configured`, `outcome`, `errorCode` |
| Steering | `firewall.steering` | Event type and `threshold` |
| Upstream error | `firewall.event` (`UPSTREAM_ERROR_RESPONSE`) | `status`, `requestId` |
| Failure | `firewall.model_routing` (`stage: "failure"`) | `code` |

**Run facts.** Untimed events are listed without an invented time: `agent.execution`, `workflow.job_result`, `model.inventory` (model count per provider and whether the pick is present), `model_endpoint.mismatch` (with `detail` and `fix`, bounded and redacted as the unified session specification requires), `model.resolution`, `model_routing.outcome` (labelled advisory), `safe_output.result` with links from `url`, and `grader.result`.

**Coverage panel.** Sources, absent components, and warnings from `session.collection`; legacy flags from MRA-001 and MRA-021; and the per-bucket count of requests with unknown cost.

### 5.8 Availability on current data

| View | Available today | Limited until |
| --- | --- | --- |
| Adoption | Yes, with `unrouted_inferred` and unknown goal for selection failures | [G-1](#g-1-model-configuration), [G-9](#g-9-engine-harness) |
| Decisions | Yes, without candidate pool | [G-2](#g-2-router-candidate-pool-and-predictions), [G-3](#g-3-router-routing-table-and-classifier-identity), [G-7](#g-7-model-inventory-rejected-for-unconfigured-endpoints) |
| Cost | Yes; sub-agent split mostly correlated or unsplit | [G-10](#g-10-request-ownership), [G-6](#g-6-baseline-price), [G-1](#g-1-model-configuration) |
| Deviations | Yes; alias moves need `model.resolution` | [G-11](#g-11-request-fields-dropped-by-the-projection), #69487 |
| Failures | Yes; upstream errors without codes | [G-5](#g-5-upstream-errors), [G-11](#g-11-request-fields-dropped-by-the-projection) |
| Outcomes | Yes, conclusion-time outcomes only | [G-4](#g-4-run-outcome) |
| Drill-down | Yes | — |

## 6. Data Gaps and Proposed Records

### 6.1 Summary

Each proposal follows the unified session specification's existing rules: the record is written by a runner or proxy component, projected by the collector with provenance (`component`, `phase`, `path`, `index`), additive to the vocabulary without changing the file-format version, and absent rather than zero when unavailable. Free-form text, bodies, headers, and content-derived hashes are excluded.

| Gap | Finding | Proposed record | Writer |
| --- | --- | --- | --- |
| [G-1](#g-1-model-configuration) Model configuration | Confirmed (not in the problem list) | New `model.config` | gh-aw runner (activation) |
| [G-2](#g-2-router-candidate-pool-and-predictions) Candidate pool and predictions | Partly recorded by AWF, dropped by the collector; predictions do not exist | Extend `firewall.model_routing` selection | Router, AWF api-proxy |
| [G-3](#g-3-router-routing-table-and-classifier-identity) Router, tables, classifier | Partly present | Extend `router` object and classification record | Router, AWF api-proxy |
| [G-4](#g-4-run-outcome) Run outcome | Conclusion-time outcomes present; delayed outcomes missing | New `safe_output.outcome` | Outcome evaluator step (runner) |
| [G-5](#g-5-upstream-errors) Upstream errors | Reach the session as a generic event without code | New `firewall.upstream_error` | AWF api-proxy |
| [G-6](#g-6-baseline-price) Baseline price | Confirmed | New `model.baseline_estimate` | Conclusion collector (runner) |
| [G-7](#g-7-model-inventory-rejected-for-unconfigured-endpoints) Inventory dropped | Confirmed; wider than reported | Collector fix to `model.inventory` | AWF (source), collector |
| [G-8](#g-8-threat-detection-attribution) Detection attribution | No gap; identified by phase | None | — |
| [G-9](#g-9-engine-harness) Engine harness | Confirmed (not in the problem list) | Extend `workflow.info` | gh-aw runner |
| [G-10](#g-10-request-ownership) Request ownership | Confirmed (not in the problem list) | Populate `subagent.request.requestId` | Engine adapters, using the AWF request ID |
| [G-11](#g-11-request-fields-dropped-by-the-projection) Dropped request fields | Confirmed (not in the problem list) | Extend `firewall.model_routing` projection | Collector |
| [G-12](#g-12-run-attempt-workflow-file-and-server) Run identity | Confirmed (not in the problem list) | Extend `workflow.info` | gh-aw runner |

### G-1. Model configuration

**Finding.** The configured `goal` and `mode` appear only on a selection record (`objective`), so runs that fail before selection have no goal or mode. `allowed-models`, `models.allowed`, `models.blocked`, and the per-run `max-ai-credits` budget appear in no session event. An unrouted run is recognizable only by the absence of routing records, which is indistinguishable from a missing source in older files (Section 4.3, order 5).

**Proposal.** A `model.config` event, emitted for every run (routed or not). The compiler passes the resolved configuration to the activation job's `generate_aw_info.cjs` step through `GH_AW_INFO_*` variables, the step writes it to `aw_info.json` as `model_config` after resolving the budget variable, and the collector projects it from the same `aw_info.json` copy and with the same provenance it uses for `workflow.info`.

| Field | Type | Meaning |
| --- | --- | --- |
| `routing.enabled` | boolean | Whether `engine.model-routing` is configured. |
| `routing.goal`, `routing.mode` | string | Configured values (`mode` may be `auto`). |
| `routing.allowedModels` | string[] | `engine.model-routing.allowed-models` as written; absent when not set. |
| `policy.allowed`, `policy.blocked` | string[] | `models.allowed` and `models.blocked` patterns; absent when not set. |
| `maxAiCredits` | number | Resolved per-run budget; `-1` when disabled. |
| `maxAiCreditsSource` | string | `frontmatter`, `variable`, or `default`. |
| `baselineModel` | string | The model the main agent would use without routing: `engine.model` when set (routing overrides it), otherwise the engine's default model. |
| `baselineModelSource` | string | `engine.model` or `engine_default`. |

Absence of the event means an older file; the page falls back to Section 4.3. Model patterns are workflow configuration already present in the compiled lock file.

### G-2. Router candidate pool and predictions

**Finding.** AWF already writes the ranked candidate list on the selection record: `ranked_choices` (an ordered list of `{model, effort}`), `eligible_choices`, and `catalogue_overlap` (for example in the AWF `model-routing/v0.28.51` record of the `copilot-budget-exhausted-steering` fixture). The collector's `firewall.model_routing` projection does not retain these fields, so only older raw session files carry them. Per-candidate predicted success and predicted credits exist nowhere at run time: the router's `/route` response carries only `ranked_choices` with `id`, `model`, and `effort`, and the router documentation states that routing tables contain only ordered model and effort identities per label cell, while prices, benchmark evidence, and fitted parameters stay outside the runtime package.

**Proposal, step 1 (collector only).** Retain on selection records:

| Field | Type | Meaning |
| --- | --- | --- |
| `rankedChoices` | `{rank, id?, model, effort?}[]` | AWF's ranked eligible choices, rank starting at 1. |
| `eligibleChoices` | number | Number of eligible choices after inventory and policy filtering. |
| `catalogueOverlap` | number | Number of routing-table choices present in the live catalogue. |

**Proposal, step 2 (router and AWF).** A routing-table schema version 2 that carries, for each label cell and choice, the table generator's `predicted_success` (probability, 0–1) and `predicted_aic` (expected AIC per run). The router returns them on each `ranked_choices` entry and adds `excluded_choices` (`{model, effort?, reason}` with `reason` in `not_in_inventory`, `outside_allowed_models`, `blocked_by_policy`, `unsupported_endpoint`, `provider_unconfigured`). AWF writes them on the selection record, and the collector projects them as `rankedChoices[].predictedSuccess`, `rankedChoices[].predictedAic`, and `excludedChoices`. The page labels predictions as router estimates and never mixes them with observed cost or outcome.

### G-3. Router, routing-table, and classifier identity

**Finding.** The selection record's `router` object carries `name` and `version`, which is the router package version (`0.1.3`), and `workflow.info.awfVersion` carries the AWF version. Neither identifies the routing tables served: gh-aw currently defaults the router image to `ghcr.io/githubnext/gh-aw-router:0.1.3` (`pkg/constants`), a workflow can replace it with `sandbox.agent.images.router`, and AWF publishes `ghcr.io/github/gh-aw-firewall/router:<awf-version>` with the package version as its version label. Tables can also be mounted separately, which is why their `schema_version` is independent of the image. The classifier's model and effort are present on classification and selection records (`classifierModel`, `classifierEffort`). Its cost is present only through accounting records with `purpose: "routing_classification"`; the classification record has no `requestId`, so attempts cannot be matched one-to-one with their cost.

**Proposal.** The router reports its table identity through `/capabilities`; AWF adds it, and the image it started, to the selection record's `router` object; and AWF adds the classifier request ID to each classification record. The collector projects the extended `router` object unchanged and adds `requestId` to classification records.

| Field | Record | Meaning |
| --- | --- | --- |
| `router.image` | Selection | Image reference with digest as started by AWF. |
| `router.tables.schemaVersion` | Selection | Routing-table `schema_version`. |
| `router.tables.profile` | Selection | Table profile used, `{goal}-{mode}`. |
| `router.tables.digest` | Selection | SHA-256 of the table file used. |
| `requestId` | Classification | AWF request ID of that classification attempt, joining its accounting record. |

Classifier cost stays on the accounting record; repeating it on the routing record would create an overlapping total (T-UAS-062).

### G-4. Run outcome

**Finding.** At conclusion time the session already carries `workflow.job_result`, `agent.execution`, `execution.result`, `detection.result`, `safe_output.result`, `safe_output.error`, and `grader.result`, and `session.collection` distinguishes "no safe-output results" from "result source missing". What is missing is the later fate of what the run produced: whether a pull request merged, an issue was closed as completed, or a change was reverted. These are unknown when the run's session file is written. gh-aw's outcome collector already classifies safe-output items later (`actions/setup/js/evaluate_outcomes.cjs`, a deterministic pre-agent step), but its results never reach a session file.

**Proposal.** A `safe_output.outcome` event. `evaluate_outcomes.cjs` writes an allowlisted JSONL file alongside its existing outputs, and the evaluating run's collector projects it with a new provenance component `outcome` and phase `workflow`, keeping it out of agent-phase scoping. Each event describes an item from an earlier run; the page joins it by subject repository and run ID and keeps the latest `evaluatedAt` per item. The event is an observation of that item's state at `evaluatedAt`, not an action of the evaluating run (T-UAS-062).

| Field | Type | Meaning |
| --- | --- | --- |
| `subject.repository`, `subject.runId`, `subject.workflow` | string, number, string | The run that produced the item. |
| `item.type`, `item.repo`, `item.number`, `item.url` | string, string, number, string | Safe-output item identity, as in `safe_output.result`. |
| `result` | string | `accepted`, `rejected`, `pending`, `ignored`, `lifecycle`, `error`, or `unknown`. |
| `evidenceStrength` | string | `strong`, `medium`, `weak`, or `none`. |
| `signal` | string | Identifier such as `merged`, `closed_without_merge`, `completed`. |
| `evaluatedAt` | string | Evaluation time. |
| `resolutionSec`, `pendingAgeSec` | number | Time to resolution or current pending age, when known. |

`detail`, comment counts tied to text, and any evaluation error text are excluded.

### G-5. Upstream errors

**Finding.** The collector walks every `.jsonl` file under `sandbox/firewall/logs/`, so AWF's `api-proxy-logs/upstream-errors.jsonl` does reach the session file: the `codex-routed-provider-unavailable` fixture's `session.collection` lists it with one event. Because no mapping exists for `UPSTREAM_ERROR_RESPONSE`, it becomes a generic `firewall.event` that keeps only `event`, `status`, and `requestId`. Provider, model, endpoint, and the provider's error code (such as `invalid_reasoning_effort`) are lost. The code exists only inside AWF's captured `response_body`, which must not be projected because provider error bodies can echo request content.

**Proposal.** AWF's api-proxy parses the provider's JSON error envelope and adds `error_code`, `error_type`, and `error_param` to each upstream error record, keeping only identifier-shaped values of at most 64 characters. The collector maps `UPSTREAM_ERROR_RESPONSE` to a new `firewall.upstream_error` event, provenance component `firewall`:

| Field | Meaning |
| --- | --- |
| `provider`, `model`, `method`, `endpoint` | Request target as recorded by AWF. |
| `status` | Upstream HTTP status. |
| `requestId` | AWF request ID, joining request and accounting records. |
| `errorCode`, `errorType`, `errorParam` | Parsed provider error identifiers. |
| `streaming` | Whether the response was streaming. |

`response_body`, response headers, upstream host and request IDs, and request tool names and counts are excluded. Older files keep the generic `firewall.event`, which the page reads as "upstream error, code unknown".

### G-6. Baseline price

**Finding.** No session event states what a routed run would have cost without routing. Comparing with a workflow's unrouted runs ([Section 5.3](#53-cost)) is observational and impossible for workflows that were never unrouted.

**Proposal.** A `model.baseline_estimate` event written by the conclusion collector, which already resolves AIC with `computeInferenceAIC` from `actions/setup/js/model_costs.cjs` for detection and eval `usage.report` events when AWF supplies none. It reprices the run's main-agent accounting records (bucket 6) at `model.config.baselineModel` prices, keeps classifier cost out (an unrouted run has no classifier), and keeps sub-agent cost at its observed value (routing does not choose sub-agent models). Provenance component `usage`, phase `conclusion`.

| Field | Meaning |
| --- | --- |
| `baselineModel`, `baselineModelSource` | From `model.config`. |
| `method` | `token_reprice`. |
| `pricingCatalog` | Identifier or digest of the pricing catalog used. |
| `rates` | Per-token AIC rates for input, output, cache read, and cache write on the baseline model. |
| `repricedRequests`, `unpricedRequests` | Coverage of the repricing. |
| `unattributedRequests` | Agent-phase requests in buckets 7 and 8, not repriced. |
| `mainAgentAic` | Repriced main-agent AIC. |
| `estimatedAic` | `mainAgentAic` plus observed sub-agent AIC. |

The event is absent when the baseline model has no known price or the run has no attributable main-agent requests. The page labels it "estimate": a different model would use different tokens, turns, and effort.

### G-7. Model inventory rejected for unconfigured endpoints

**Finding.** Confirmed, and wider than reported. `normalizeModelInventoryData` in `actions/setup/js/unified_session_payload.cjs` rejects the whole snapshot when any endpoint has `routing_models: null`, so the collector reports `invalid_model_inventory` and emits no `model.inventory` event, losing the configured endpoints too. Separately, an endpoint with `models: null` and a boolean `configured` is projected as `models: []` without `configured`, so an unconfigured endpoint looks like a configured endpoint with no models.

**Proposal.** A collector fix to the existing `model.inventory` event, which AWF already writes through `/reflect`. Per endpoint, retain `configured` when supplied; treat `routing_models: null` as absent routing metadata for that endpoint; and omit `models` when the source is `null`. Also retain each routing model's `supported_reasoning_efforts` as `supportedReasoningEfforts`, so the page can show which efforts were live. Readers treat an absent `models` or `routingModels` as unknown.

### G-8. Threat detection attribution

**Finding.** Threat-detection cost is not identified by `firewall.token_usage.purpose`; AWF writes no detection purpose. It is identified by provenance phase `detection`, which the collector assigns to all sources under `threat-detection/`. No record change is needed; MRA-070 and MRA-071 encode the rule.

### G-9. Engine harness

**Finding.** `workflow.info.engineId` is `copilot` for both Copilot CLI and Copilot SDK (`engine.copilot-sdk: true`). `aw_info.json`'s `agent_runtime` is the sandbox runtime, not the harness. `subagent.model_unavailable` is written only by the SDK driver, but its absence does not imply the CLI.

**Proposal.** The compiler sets `GH_AW_INFO_ENGINE_HARNESS` (`copilot-cli`, `copilot-sdk`, or the engine ID for others), `generate_aw_info.cjs` writes `engine_harness`, and the collector projects it as `workflow.info.engineHarness`. Absent means unknown.

### G-10. Request ownership

**Finding.** In every golden fixture, AWF records `purpose` only for classifier requests; main-agent and sub-agent requests have none. `subagent.request` declares a `requestId` field, but no adapter fills it (the Pi fixture's records have none), so per-agent cost depends on matching models, token counts, and order. AWF already returns its request ID to the client in the `x-request-id` response header.

**Proposal.** Engine adapters and the Copilot SDK driver record the AWF `x-request-id` response header as `subagent.request.requestId` for each sub-agent request, and AWF sets `purpose: "agent"` or `"subagent"` when the harness identifies the caller. The join keeps credits and served models proxy-sourced while the definition name stays engine-reported, as T-UAS-073 allows.

### G-11. Request fields dropped by the projection

**Finding.** AWF request and failure records carry fields that the current projection drops but older raw files keep: request `status` (HTTP status), `unavailable` (models AWF found unavailable for that request), `requested_endpoint`, `selected_endpoint`, `upstream_endpoint`, and failure `phase` (for example `primary`).

**Proposal.** The collector retains them as `status`, `unavailable`, `requestedEndpoint`, `selectedEndpoint`, `upstreamEndpoint`, and `phase` on `firewall.model_routing`, extending the field list in the unified session specification's "Model routing attribution" requirement.

### G-12. Run attempt, workflow file, and server

**Finding.** `workflow.info.workflow` is the display name, which can change or collide across files; the run attempt and server URL are not projected, although `aw_info.json` already records `run_attempt`.

**Proposal.** `generate_aw_info.cjs` also writes `workflow_ref` (from `GITHUB_WORKFLOW_REF`) and `server_url` (from `GITHUB_SERVER_URL`), and the collector projects `runAttempt`, `workflowRef`, and `serverUrl` on `workflow.info`. The page then keys workflows by `workflowRef` path, deduplicates attempts exactly, and builds the Actions link from `serverUrl`.

## 7. Rollout

1. Build the page on current data with the rules in Sections 3–5. Every view in [Section 5.8](#58-availability-on-current-data) works with the stated limits.
2. Collector-only changes: G-2 step 1, G-7, G-11, and G-12.
3. gh-aw runner changes: G-1, G-9, and G-6 (after G-1).
4. AWF and router changes: G-2 step 2, G-3, G-5, and the AWF side of G-10; engine adapters for G-10.
5. Outcome evaluator: G-4.

Each step adds fields or event types only, so files produced before it continue to load (MRA-001, MRA-003, MRA-020).

## 8. Open Questions

1. Should `N_min` and the comparison window be per page or per view?
2. Should attempts of one run be merged, or should only the final attempt count once G-12 lands?
3. Should G-2's predictions be per label cell (as tables are generated) or adjusted by the router per request?
4. Should `safe_output.outcome` events from the outcome collector be ingested from that workflow's session files only, or should every repository run its own evaluator?
5. Does any engine other than Copilot need a harness value in G-9?

## 9. References

- [Unified Agent Session Specification](/gh-aw/specs/unified-agent-session-specification/), version 1.10.0: Sections 3.5, 4.7, 4.8, and 7.2; requirements T-UAS-062, T-UAS-063, T-UAS-064, T-UAS-070, T-UAS-071, and T-UAS-073.
- [AI Credits Specification](/gh-aw/specs/ai-credits-specification/).
- [Model Alias Specification](/gh-aw/specs/model-alias-specification/).
- [Graders Specification](/gh-aw/specs/graders-specification/).
- [Model routing reference](/gh-aw/reference/model-routing/).
- Pull request #69487, which proposes the `model.resolution` event.
- AWF router: `containers/router/README.md` and `containers/router/openapi.yaml` in `github/gh-aw-firewall`.
- Golden fixtures: `pkg/cli/testdata/model_routing_golden/`.

## 10. Change Log

### Version 0.1.0 — Draft (2026-10-16)

- Initial design: views, metric definitions with source events and fields, reading rules, and proposed records for twelve data gaps.
