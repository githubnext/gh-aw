---
title: GitHub Work Item Impact Score Specification
description: Formal W3C-style specification for numeric GitHub work item impact scoring, attribution to humans, agents, workflows, and components, cost ranking, artifacts, and dashboard behavior.
version: 0.1.0
status: Working Draft
date: 2026-06-25
last_updated: 2026-06-25
editors:
  - GitHub gh-aw Team
---

# GitHub Work Item Impact Score Specification

**Version**: 0.1.0
**Status**: Working Draft
**Publication Date**: June 25, 2026
**Latest Version**: https://github.com/github/gh-aw/blob/main/specs/impact-score-spec.md
**Editors**: GitHub gh-aw Team

---

## Abstract

This specification defines Impact Score: a numeric, repository-configurable score for GitHub work items such as issues and pull requests. Impact Score makes explicit a prioritization signal that developers, maintainers, managers, and agents have historically inferred implicitly when selecting work. It specifies the typed repository knowledge graph model, the safe `aw.json` impact policy contract, history-derived policy bootstrap, attribution to humans, agents, workflows, and components, AIC cost joining, generated artifact contract, browser dashboard, security controls, and conformance tests for the experimental `pkg/impactscore` package, importable runner, and hidden `gh aw impact` command.

## Status of This Document

This document is a **Working Draft** maintained by the GitHub `gh-aw` Team. It may be changed, replaced, or made obsolete by subsequent versions.

Version 0.1.0 documents an experimental implementation of a broader GitHub platform concept. The implementation is exposed through the hidden `gh aw impact` command while the scoring model, attribution model, dashboard, and artifact contract are still being evaluated.

Sections explicitly marked **Informative** are non-normative. All other sections are normative unless stated otherwise.

Implementations claiming conformance MUST identify the version and conformance classes they implement.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Terminology](#3-terminology)
4. [Architecture](#4-architecture)
5. [Data Model](#5-data-model)
6. [Impact Score Model](#6-impact-score-model)
7. [Attribution and Cost Ranking](#7-attribution-and-cost-ranking)
8. [Artifact Contract](#8-artifact-contract)
9. [Dashboard Contract](#9-dashboard-contract)
10. [Security and Privacy](#10-security-and-privacy)
11. [Error Handling](#11-error-handling)
12. [Compliance Testing](#12-compliance-testing)
13. [Examples](#13-examples)
14. [References](#14-references)
15. [Change Log](#15-change-log)

---

## 1. Introduction

### 1.1 Purpose

Impact Score helps repository owners assign an explicit numeric score to GitHub work items. It turns repository-specific intuition about “important work” into a configurable, inspectable, and automatable signal.

Developers already make implicit impact judgments when they choose which issue to pick up, which pull request to review, which incident to remediate, which customer escalation to prioritize, or which automation to fund. Impact Score makes that implicit judgment explicit so it can be queried, compared, attributed, tuned, and audited.

The feature also helps repository owners understand which workflows, agents, teams, components, services, or campaigns are connected to high-impact GitHub work and how that impact compares with observed costs.

The feature is intended to answer these questions:

- Which issues and pull requests appear most important according to repository-specific evidence?
- Which workflows, agents, teams, components, services, or campaigns are connected to high-impact work?
- Which workflows or agents have high impact score and low AIC cost?
- Which workflows or agents have high impact score but high AIC cost and SHOULD be optimized?
- Which sources of work have impact but missing cost data?
- Which repository-specific labels, dimensions, metrics, and graph edges SHOULD affect scoring?

### 1.2 Scope

This specification covers:

- normalized GitHub issue and pull request work items;
- typed repository knowledge graph construction from work item fields, dimensions, nodes, and edges;
- safe `aw.json` impact policy calculation;
- attribution of impact score to humans, agents, workflows, components, services, teams, and campaigns;
- joining workflow impact with observed AIC cost;
- JSON and selected text or HTML report artifact names and contents;
- the dashboard user interface, including workflow charts and high-impact work item links;
- security, privacy, reliability, and conformance requirements.

This specification does not define:

- a universal business-value formula;
- billing-accurate AIC cost accounting;
- production service deployment requirements;
- a requirement that `impact-score` be part of the public `gh-aw` CLI;
- a requirement that Impact Score be exposed in the GitHub product UI;
- a requirement to use a graph database, global ontology, semantic reasoning engine, or cross-repository identity system;
- vendor-specific analytics dashboards.

### 1.3 Design Goals

The design goals are:

1. **GitHub-native semantics**: Use GitHub issues, pull requests, labels, assignees, authors, reviews, workflow provenance, and cost logs as the primary surface.
2. **Repo-specific configurability**: Allow each repository to define what impact means through Notepad-editable JSON rules in `.github/workflows/aw.json` that are parsed into a typed internal policy representation before scoring.
3. **Explainability**: Make impact score inputs visible through artifacts and dashboard controls.
4. **Safe interactivity**: Static HTML MUST NOT attempt to execute commands or expose local score-policy command controls.
5. **Scriptability**: Preserve machine-readable JSON artifacts for automation.
6. **Progressive adoption**: Provide a transparent generated `aw.json` impact policy and observed-history bootstrap before requiring hand-written policy tuning.
7. **Attribution flexibility**: Allow impact to be attributed to human work, agent work, workflows, components, teams, and campaigns without changing the core work-item score.
8. **Product portability**: Keep the model suitable for possible future GitHub-native surfaces such as sortable or filterable issue and pull request fields.
9. **Value/cost pairing**: Present observed automation cost beside attributed impact so spend can be evaluated as investment, optimization opportunity, or missing evidence rather than as an isolated number.

### 1.4 Design Rationale (Informative)

Impact Score derives from early work on workflow portfolio management: treating agentic workflows as a portfolio of investments that should be compared by the work they create, the outcomes they influence, and the operating cost they consume.

The design intentionally builds on the strength of GitHub itself. GitHub already contains the repository-local work graph: issues, pull requests, labels, reviews, milestones, workflow runs, generated issue provenance, comments, and artifacts. Impact Score uses those native signals as the measurement substrate instead of requiring a separate planning system or external portfolio database.

The cost model is intentionally paired with value. AIC cost and action minutes can feel daunting when viewed alone; side-by-side impact and cost views help repository owners distinguish workflows that should be scaled, workflows that should be optimized, workflows that need better cost evidence, and workflows that may not justify continued investment.

---

## 2. Conformance

### 2.1 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.2 Conformance Classes

This specification defines the following conformance classes:

| Class | Responsibility |
|---|---|
| **Core Library** | Implements graph construction, feature extraction, item ranking, attribution, and workflow ranking. |
| **Impact Runner** | Fetches or normalizes repository data, optionally generates an `aw.json` policy, writes artifacts, and optionally writes the dashboard. |
| **Dashboard UI** | Presents workflow impact/cost charts and linked high-impact work items. |
| **Artifact Consumer** | Reads generated JSON, text, or HTML artifacts without depending on internal Go APIs. |
| **Attribution Consumer** | Consumes work item impact scores and attributes them to users, agents, workflows, components, campaigns, or other repository entities. |

An implementation MAY conform to one or more classes. It MUST state each claimed class.

### 2.3 Compliance Levels

| Level | Name | Requirements |
|---|---|---|
| **Level 1** | Core Impact Scoring | Core Library data model, graph construction, feature extraction, item scoring, attribution, and artifact generation. |
| **Level 2** | Interactive Dashboard | Level 1 plus dashboard rendering and high-impact work item links. |

A claim of a compliance level MUST satisfy every applicable MUST and MUST NOT requirement for all claimed conformance classes at that level.

---

## 3. Terminology

### 3.1 Impact Score

An **impact score** is a numeric, unitless heuristic score estimating how important a GitHub work item appears according to the selected model. An impact score is not money, quality, merge priority, or a billing amount.

When impact score is shown for a workflow, agent, human, component, team, campaign, or other entity, that value is an attribution or aggregation of work item impact scores. The work item score is the canonical score.

### 3.2 Work Item

A **work item** is a normalized GitHub issue or pull request represented by a `WorkItem` record.

### 3.3 Dimension

A **dimension** is a categorical work-item signal such as label, component, top-level area, risk domain, customer impact, team, service, or work type.

### 3.4 Measure

A **measure** is a numeric work-item signal such as changed files, sensitive path count, review comments, saved review minutes, or customer accounts.

### 3.5 Workflow Attribution

**Workflow attribution** is the process of assigning item impact score to one or more source workflows linked to the item.

### 3.6 AIC Cost

**AIC cost** is the observed model or AI cost associated with a workflow run. This specification treats AIC cost as an input signal and does not define provider billing semantics.

### 3.7 Actor

An **actor** is a source of work or work execution, including a human user, GitHub App, Copilot cloud agent, automation account, agentic workflow, bot, or other identifiable entity.

### 3.8 Component

A **component** is a repository, service, package, directory, subsystem, product area, or other technical or organizational unit to which work can be attributed.

### 3.9 Campaign

A **campaign** is a coordinated collection of work items pursuing a shared objective, such as reducing flaky tests, remediating a vulnerability class, migrating APIs, improving documentation, or optimizing cost.

### 3.10 Native Work Item Field

A **native work item field** is a field surfaced directly on GitHub issues, pull requests, project items, or related GitHub UI surfaces. This specification does not require native product integration, but it reserves compatibility expectations for future implementations.

### 3.11 Typed Repository Knowledge Graph

A **typed repository knowledge graph** is a graph whose nodes represent GitHub work items, repository signals, actors, workflows, components, campaigns, and other repository-local entities, and whose edges represent typed relationships among them.

This term is intentionally scoped to repository-local work knowledge. It does not imply a graph database, a global ontology, semantic reasoning, or cross-repository identity resolution.

---

## 4. Architecture

### 4.1 Overview

The Impact Score feature has the following processing flow:

1. The Impact Runner fetches GitHub issues and pull requests for a repository, defaulting to the current git checkout's GitHub repository when no explicit repository is supplied.
2. The Impact Runner normalizes each issue or pull request into a `WorkItem`.
3. The Core Library builds a typed repository knowledge graph.
4. The Impact Runner MAY generate a history-derived `aw.json` impact policy.
5. The Impact Runner parses and validates `aw.json` impact rules into a typed internal policy representation.
6. The Core Library ranks work items from release-note importance, observed score inputs, or `aw.json` policy output.
7. The Core Library or an Attribution Consumer attributes work item impact score to workflows, actors, components, services, teams, or campaigns.
8. The Impact Runner joins workflow or agent impact with optional cost logs.
9. The Impact Runner writes JSON artifacts and the selected report artifact format.
10. The Dashboard UI displays workflow charts and high-impact work items.

The hidden `gh aw impact` command uses GitHub APIs for current repository work-item data and agentic workflow definitions discovered from `.github/workflows/*.lock.yml` or `.github/workflows/*.lock.yaml`, delegates workflow run and cost observations to `gh aw logs --json`, and treats local output files as normalized cache artifacts rather than as the primary source of truth.

### 4.2 Components

| Component | Implementation Path | Role |
|---|---|---|
| Core Library | `pkg/impactscore` | Typed repository knowledge graph, features, item ranks, and attribution inputs. |
| Impact Runner | `pkg/impactscore/runner` | Repository fetch, `aw.json` policy bootstrap/evaluation, artifact writing, and selected report rendering. |
| Hidden CLI Command | `gh aw impact` | Hidden experimental entry point for the Impact Runner. |
| Dashboard | `impact_score_dashboard.html` | Static browser dashboard. |
| Attribution Consumer | external or future built-in consumers | Assigns work item impact score to actors, components, campaigns, workflows, or product surfaces. |

---

## 5. Data Model

### 5.1 WorkItem

An implementation conforming to the Core Library class MUST support a work item model with at least the following fields:

| Field | Type | Requirement |
|---|---|---|
| `Repo` | string | SHOULD identify the owner/repository when known. |
| `Number` | integer | MUST identify the GitHub issue or pull request number. |
| `Type` | string | MUST be `issue`, `pr`, or another documented item type. |
| `Title` | string | SHOULD contain the GitHub title. |
| `State` | string | SHOULD contain the coarse GitHub state, such as `open` or `closed`. |
| `StateReason` | string | MAY contain normalized lifecycle detail beyond coarse state, such as `completed`, `not_planned`, `draft`, `merged`, or `closed_unmerged`. |
| `Labels` | string array | MAY contain GitHub labels. |
| `Author` or equivalent actor signal | string | MAY identify the user, app, bot, or agent that opened or executed the work. |
| `Dimensions` | map of string to string array | MAY contain repo-specific categorical signals. |
| `Measures` | map of string to number | MAY contain repo-specific numeric signals. |
| `SourceWorkflows` | string array | MAY contain workflow names linked to the item. |
| `GraphNodes` | node array | MAY add arbitrary graph nodes. |
| `GraphEdges` | edge array | MAY add arbitrary graph edges. |

### 5.2 Typed Repository Knowledge Graph Nodes and Edges

The typed repository knowledge graph model MUST support typed nodes and typed edges. A graph edge with an empty source SHOULD be treated as an edge from the work item that contributed it.

Built-in graph construction SHOULD include nodes and edges for labels, components, areas, context signals, workflow provenance, actor provenance, architecture references, release categories, and change types when those signals are available.

Implementations MAY store the typed repository knowledge graph in memory, JSON artifacts, a graph database, or another representation. Conformance MUST NOT require a graph database, global ontology, semantic reasoning engine, or cross-repository identity system.

### 5.3 Features

Feature extraction MUST produce dimensions and measures for a work item. Implementations MUST preserve repo-defined `WorkItem.Dimensions` and `WorkItem.Measures` so policy evaluation and artifact consumers can use them.

When `WorkItem.StateReason` is present, feature extraction SHOULD expose it through the standard `state_reason` dimension. Implementations MUST NOT overload `State` with issue close reasons or pull request outcomes that are not GitHub issue-list states.

### 5.4 Attribution Surfaces

Implementations SHOULD support attribution surfaces independent of the core work item score. Attribution surfaces MAY include:

- human authors, assignees, reviewers, or maintainers;
- Copilot cloud agents, GitHub Apps, or bots;
- agentic workflows or GitHub Actions workflows;
- repository components, services, directories, or packages;
- teams, owners, milestones, projects, or campaigns.

An attribution surface MUST NOT change the canonical work item impact score solely because the target entity changed. Attribution surfaces consume and aggregate item scores.

---

## 6. Impact Score Model

### 6.1 Policy-Based Impact

Impact score SHOULD be calculated by a repository scoring policy over normalized item features. The policy source MUST be visible to repository owners when it contributes non-zero scores. Implementations MUST NOT inject hidden starter scores into the policy input.

The default repository scoring policy MUST be stored under the `impact` key in `.github/workflows/aw.json`. The Impact Runner MUST parse the JSON policy into a typed internal representation before evaluation. The human-editable JSON file is a serialization format; it MUST NOT be treated as executable code.

Generated init commands MAY include a visible editable baseline policy so first-run output contains impact scores before the repository owner writes a fully custom policy. Such baseline logic MUST be emitted into `.github/workflows/aw.json` and MUST be editable by the repository owner.

When no explicit score policy is supplied, an implementation MAY use observed release metadata, such as `release_note_importance`, as a minimal default score. Items without explicit policy scores or observed release scores SHOULD remain unscored rather than receiving implicit heuristic values.

### 6.2 Observed Item Score Inputs

Observed score inputs are repository facts that represent previously assigned impact. Implementations MAY recognize observed numeric item measures such as `impact_score` or `score`, and MAY recognize positive `release_note_importance` for released items. Observed values MUST be clamped to the `0-10` range before they are used for ranking or bootstrap learning.

Observed score inputs MUST be distinguished from AIC cost. AIC cost measures agent or workflow spend and MUST NOT be treated as item impact.

### 6.3 `.github/workflows/aw.json` Impact Policy Contract

The repository impact policy MUST be represented as JSON at `impact` in `.github/workflows/aw.json`. The `impact` object MUST contain a `rules` array when policy rules are defined. The `impact` object MAY contain `version`, `base`, and `clamp` fields.

The following JSON object is a conforming minimal policy shape:

```json
{
  "impact": {
    "version": 1,
    "base": 1,
    "clamp": { "min": 0, "max": 10 },
    "rules": [
      {
        "name": "ignore duplicate work",
        "when": { "any_label": ["duplicate", "invalid", "wontfix"] },
        "score": 0,
        "stop": true
      },
      {
        "name": "ignore non-delivered closed work",
        "when": { "any_dimension": { "state_reason": ["not_planned", "closed_unmerged"] } },
        "score": 0,
        "stop": true
      },
      {
        "name": "security work",
        "when": {
          "any": [
            { "any_label": ["security"] },
            { "any_signal": ["security", "vulnerability", "credential"] }
          ]
        },
        "min": 7
      },
      {
        "name": "sensitive path boost",
        "when": { "measure_gt": { "sensitive_path_count": 0 } },
        "add": 1
      }
    ]
  }
}
```

Implementations MUST validate `impact.rules` before scoring. Validation MUST reject unknown operation names, non-numeric score operations, invalid clamp bounds, unsupported condition keys, and rules that contain no operation. Validation errors SHOULD identify the rule name or array index.

### 6.4 Policy Internal Representation

The Impact Runner MUST translate the JSON policy into an internal representation with explicit policy, rule, condition, and operation nodes. The internal representation owns policy semantics, including rule order, matching behavior, stop behavior, and score clamping.

A conforming internal representation MUST support these rule operations:

| Operation | Meaning |
|---|---|
| `score` | Set the current score to the supplied numeric value. |
| `min` | Raise the current score to at least the supplied numeric value. |
| `max` | Lower the current score to at most the supplied numeric value. |
| `add` | Add the supplied numeric value to the current score. |
| `stop` | Stop evaluating subsequent rules after the current matching rule. |

A conforming internal representation SHOULD support these condition keys:

| Condition | Meaning |
|---|---|
| `any_label` | Match when the item has at least one listed label. |
| `all_label` | Match when the item has every listed label. |
| `any_signal` | Match when the item has at least one listed context signal. |
| `all_signal` | Match when the item has every listed context signal. |
| `any_component` | Match when the item touches at least one listed component. |
| `any_area` | Match when the item touches at least one listed area. |
| `any_source_workflow` | Match when the item is linked to at least one listed source workflow. |
| `any_dimension` | Match when each named dimension has at least one listed value. |
| `all_dimension` | Match when each named dimension has every listed value. |
| `measure_gt` | Match when each named measure is greater than the supplied value. |
| `measure_gte` | Match when each named measure is greater than or equal to the supplied value. |
| `measure_lt` | Match when each named measure is less than the supplied value. |
| `measure_lte` | Match when each named measure is less than or equal to the supplied value. |
| `state` | Match when the item state is one of the listed values. |
| `type` | Match when the item type is one of the listed values. |

When a condition object contains multiple condition keys, all keys in that object MUST match unless they are placed inside an explicit `any` group. Implementations MAY support nested `all`, `any`, and `not` condition groups.

Policy evaluation MUST be deterministic for the same input item and policy. The final score MUST be clamped to the configured clamp range, or to `0-10` when no clamp is configured.

### 6.5 Score Explanation

Policy-derived work item ranks MUST include structured score explanation data. The explanation SHOULD identify the policy path, policy version, policy digest, and matched rule names that contributed to the final score.

The following JSON object is a conforming score explanation shape:

```json
{
  "score_explanation": {
    "policy_path": ".github/workflows/aw.json",
    "policy_version": 1,
    "policy_sha256": "...",
    "matched_rules": ["security work", "sensitive path boost"]
  }
}
```

The policy digest SHOULD be computed from the normalized `impact` object rather than from unrelated repository configuration fields. Implementations MAY omit explanation fields that are not applicable, such as policy fields for observed release-note scores.

### 6.6 History Bootstrap

The Impact Runner SHOULD support initializing an `.github/workflows/aw.json` impact policy from observed repository history. The bootstrap SHOULD:

- start from explicit JSON policy rules under `impact.rules`;
- consider closed issues, merged or closed pull requests, and workflow-created issues that have observed item impact scores;
- mine simple predicates over labels, context signals, components, areas, source workflows, and built-in measures;
- include explicit metadata fields that show support count, mean observed impact, and observed repository baseline when observed scores exist;
- keep only conservative, simple rules with sufficient support;
- write JSON that the repository owner can edit directly in a plain text editor.

The reference Impact Runner bootstrap command MUST target `.github/workflows/aw.json`; it MUST NOT accept arbitrary impact policy output paths.

Because JSON does not support comments, implementations SHOULD store explanatory bootstrap metadata in explicit fields such as `description`, `evidence`, `support`, `mean`, and `baseline` rather than relying on comments.

History bootstrap MUST NOT be treated as a final or universal score model. It is an editable first draft of the repository-specific impact model.

---

## 7. Attribution and Cost Ranking

### 7.1 Canonical Attribution Formula

Impact score MAY be attributed to any linked entity. If an item is linked to `N` entities in the same attribution class, each entity receives `item impact score / N` unless the implementation documents a different weighting model.

```text
entity impact score = sum(item impact score / count(item linked entities in class))
```

Items with no linked entity in an attribution class MUST NOT contribute to that class's aggregate impact score.

Work items and pull requests with no linked agentic workflow MUST remain visible as work items in item artifacts and high-impact work item report surfaces. They MUST NOT be folded into agentic workflow rankings unless an explicit agentic workflow link is present.

### 7.2 Workflow Attribution

Workflow impact score is a specialization of entity attribution. If an item is linked to `N` source workflows, each workflow receives `item impact score / N` unless a repository-specific attribution model overrides this behavior.

```text
workflow impact score = sum(item impact score / count(item source workflows))
```

### 7.3 Human, Agent, Component, and Campaign Attribution

Implementations MAY compute impact scores for humans, agents, components, services, teams, or campaigns. These attributions SHOULD use the same canonical work item scores as workflow attribution. They MAY use different relationship classes, such as author, assignee, reviewer, workflow provenance, touched component, project field, milestone, label, or custom graph edge.

Human and agent attribution SHOULD distinguish between creating work, implementing work, reviewing work, and automating work when the data supports that distinction.

### 7.4 Cost Join

The Impact Runner MAY load workflow or agent cost observations from `gh aw logs --json`. Cost records SHOULD include workflow or agent name, run ID, AIC cost, token usage, turns, action minutes, error count, and source path when available.

Duplicate cost runs with the same workflow and run ID SHOULD be deduplicated, preserving the record with the highest AIC cost or richest token usage.

### 7.5 Derived Entity Metrics

The workflow rank model MUST compute the following for workflow entities, and other attribution consumers SHOULD compute analogous fields when applicable:

- total attributed impact score;
- linked item count;
- open item count;
- released item count;
- run count;
- total AIC cost;
- average AIC cost per run when run count is non-zero;
- impact score per AIC when total AIC cost is positive;
- AIC per impact score when impact score is positive.

### 7.6 Action Zones

Workflow action zones SHOULD classify workflows using median impact and median cost thresholds:

| Zone | Meaning |
|---|---|
| `keep / scale` | High impact score and lower observed cost. |
| `optimize` | High impact score and high observed cost. |
| `waste review` | Low impact score and high observed cost. |
| `needs cost` | Positive impact score and missing observed cost. |
| `monitor` | Low impact score and lower observed cost. |

---

## 8. Artifact Contract

### 8.1 Output Directory

When no output directory is provided, the Impact Runner SHOULD write to a unique directory under the `gh-aw/impact-score` subdirectory of the system temporary directory. When durable cache artifacts are desired, callers SHOULD provide `--out` with an explicit path.

The Impact Runner SHOULD infer the target repository from the current checkout.

### 8.2 Required JSON Artifacts

The Impact Runner MUST write the following JSON artifacts:

| Artifact | Contents |
|---|---|
| `summary.json` | Repository, generation time, and artifact counts. |
| `items.json` | Normalized `WorkItem` records. |
| `workflows.json` | Agentic workflow definitions discovered from `.github/workflows/*.lock.yml` or `.github/workflows/*.lock.yaml` in the target GitHub repository. |
| `cost_runs.json` | Normalized workflow cost runs from `gh aw logs --json` and AIC observations parsed from issue or pull request text when available. |
| `item_ranks.json` | Ranked work items, including impact score source and structured score explanation when available. |
| `features.json` | Per-item extracted dimensions and measures. |
| `workflow_ranks.json` | Workflow impact-score/cost rankings. |
| `graph_nodes.json` | Typed graph nodes. |
| `graph_edges.json` | Typed graph edges. |

### 8.3 Report Artifacts

The Impact Runner MUST default to text report rendering when no report format is provided. The default text report MUST be written to:

```text
impact_score_report.txt
```

The Impact Runner MAY support `--report-format text` and `--report-format html`.

### 8.4 HTML Artifact

When `--report-format html` is selected, the Impact Runner MUST write:

```text
impact_score_dashboard.html
```

---

## 9. Dashboard Contract

### 9.1 Visible Views

The dashboard MUST provide a **Workflows** view and a **Work Items** view. It MUST NOT require a separate visual scoring-rule builder tab or separate scoring tab.

The top-level page context SHOULD show the repository as the title and the generation timestamp as the subtitle.

### 9.2 Workflows View

The Workflows view MUST contain:

- summary cards for total impact score, total cost signal, and workflow count, with workflow count allowed to span multiple grid cells;
- a workflow impact score versus cost signal chart;
- a workflow impact score ranking chart;
- action-zone legend or equivalent explanation.

When AIC cost is available, the cost signal SHOULD be AIC. When AIC cost is unavailable but action minutes are available, the dashboard MAY use action minutes as the visible cost signal and MUST label the axis accordingly.

### 9.3 Work Items View

The Work Items view MUST show scored work items with no linked agentic workflow. These items SHOULD include human-created pull requests, human-created issues, and other work that is not attributed to an agentic workflow. Other Work items MUST remain excluded from workflow rankings unless an explicit agentic workflow link is present.

The Work Items view SHOULD also show a separate filterable table for work items linked to agentic workflows, adjacent to the unlinked work table when viewport width permits.

The Work Items view MUST NOT attempt to execute local commands from the browser.

### 9.3.1 Future Native GitHub UI Integration

Future implementations MAY expose impact score as a native or custom field on GitHub issues, pull requests, Projects items, or repository dashboards. Such a field SHOULD be sortable and filterable. If exposed, the UI SHOULD make clear which score policy or observed score input produced the score.

Native UI integrations MUST preserve the distinction between canonical work item impact score and attributed aggregate impact score.

### 9.4 Policy Commands

Impact policy bootstrap and rerun commands SHOULD remain Impact Runner command-line workflows, not dashboard controls. A bootstrap command SHOULD initialize `impact.rules` in `.github/workflows/aw.json`. A policy rerun command SHOULD read `.github/workflows/aw.json` and evaluate the typed internal policy representation.

The dashboard MUST NOT run, copy, display, or download local command controls by default.

### 9.5 Scored Work Items

The dashboard SHOULD include a compact scored work item table. Work item entries SHOULD link to their GitHub issue or pull request URL when repository and item number are available. Links MUST use `rel="noopener noreferrer"` when they open in a new tab.

The scored work item table SHOULD show scores from the current run and SHOULD be filterable by item title, number, type, state, state reason, score source, matched rule, and source workflow. Items with policy-derived scores SHOULD show a compact policy and matched-rule explanation when available. Items with no linked agentic workflow SHOULD be labeled as such and also appear in the Other Work view rather than being hidden from the dashboard.

### 9.6 Static HTML Behavior

When the dashboard is opened as static HTML, it MUST NOT attempt to execute shell commands or expose local command controls.

---

## 10. Security and Privacy

### 10.1 GitHub Data

Implementations MUST treat fetched GitHub issue and pull request data as repository data subject to repository visibility and access controls.

### 10.2 Tokens

The Impact Runner MAY use `GITHUB_TOKEN` or `gh auth token` for GitHub API access. It MUST NOT write tokens into artifacts, dashboards, or logs.

### 10.3 Browser Execution Boundary

Static dashboard HTML MUST NOT execute local commands. Rerun support MUST be mediated through explicit terminal execution of a generated command.

### 10.4 Links

Dashboard links to GitHub work items SHOULD open in a new tab using `rel="noopener noreferrer"`.

---

## 11. Error Handling

### 11.1 Fetch Errors

The Impact Runner MUST return a non-zero exit status when required GitHub API requests fail.

### 11.2 Impact Policy Errors

Invalid `aw.json` impact policy syntax, schema, condition keys, operation keys, or operation values MUST prevent scoring with that policy and SHOULD produce an actionable error message. The Impact Runner SHOULD include the rule name or rule index when available.

---

## 12. Compliance Testing

### 12.1 Test Suite Requirements

Conforming implementations SHOULD provide automated tests for each claimed conformance class.

### 12.2 Required Tests

| Test ID | Requirement | Procedure | Expected Result |
|---|---|---|---|
| **T-IS-001** | Artifact names | Run the Impact Runner with default options and with each supported `--report-format` value. | The default run writes `impact_score_report.txt`; `html` writes `impact_score_dashboard.html`. |
| **T-IS-002** | Package import path | Build the command and package. | Imports resolve through `pkg/impactscore`. |
| **T-IS-003** | Workflow attribution | Rank an item linked to multiple workflows. | Item score is divided across linked workflows. |
| **T-IS-004** | Missing cost zone | Rank a workflow with positive impact and no cost. | Workflow action zone is `needs cost`. |
| **T-IS-005** | Dashboard work item links | Render dashboard with work item data. | Scored work item table entries link to GitHub issue or pull request URLs with `noopener noreferrer`. |
| **T-IS-006** | Static dashboard boundary | Open the generated dashboard. | UI does not attempt direct command execution and does not expose local command controls. |
| **T-IS-007** | Dashboard JavaScript | Extract generated dashboard script and run syntax validation. | Script parses successfully. |
| **T-IS-008** | Attribution separation | Attribute the same work item to two different entity classes. | Canonical item impact score remains unchanged while aggregate scores differ by relationship class. |
| **T-IS-009** | History bootstrap | Run the Impact Runner policy initialization flow. | `.github/workflows/aw.json` contains `impact.rules` with history-derived rule evidence or baseline rules. |
| **T-IS-010** | JSON policy execution | Run the Impact Runner with `.github/workflows/aw.json` impact rules. | Policy-derived item scores are used in `item_ranks.json` and workflow attribution. |
| **T-IS-011** | Policy validation | Run the Impact Runner with an invalid `impact.rules` entry. | Scoring fails before ranking and reports the invalid rule name or index. |
| **T-IS-012** | Score explanation | Run the Impact Runner with `.github/workflows/aw.json` impact rules that match a work item. | `item_ranks.json` includes policy path, policy version, policy digest, and matched rule names for the scored item. |
### 12.3 Compliance Checklist

| Requirement Area | Test IDs | Level |
|---|---|---|
| Core library and package naming | T-IS-002, T-IS-003, T-IS-004, T-IS-008 | 1 |
| Artifact contract | T-IS-001 | 1 |
| Dashboard behavior | T-IS-005, T-IS-006, T-IS-007 | 2 |
| JSON impact policies | T-IS-009, T-IS-010, T-IS-011, T-IS-012 | 1 |

---

## 13. Examples

### 13.1 Generate Static Dashboard

```bash
gh aw impact --report-format html
```

### 13.2 `.github/workflows/aw.json` Impact Policy

```json
{
  "impact": {
    "version": 1,
    "base": 1,
    "clamp": { "min": 0, "max": 10 },
    "rules": [
      {
        "name": "ignore duplicate work",
        "when": { "any_label": ["duplicate", "invalid", "wontfix"] },
        "score": 0,
        "stop": true
      },
      {
        "name": "ignore non-delivered closed work",
        "when": { "any_dimension": { "state_reason": ["not_planned", "closed_unmerged"] } },
        "score": 0,
        "stop": true
      },
      {
        "name": "security work",
        "when": {
          "any": [
            { "any_label": ["security"] },
            { "any_signal": ["security", "vulnerability", "credential"] }
          ]
        },
        "min": 7
      },
      {
        "name": "broad sensitive change",
        "when": {
          "measure_gt": { "sensitive_path_count": 0 },
          "measure_gte": { "component_count": 3 }
        },
        "add": 2
      }
    ]
  }
}
```

### 13.3 Initialize Missing Scoring Policy

```bash
gh aw impact \
  --report-format html
```

### 13.4 Rerun With Edited `.github/workflows/aw.json` Policy

```bash
gh aw impact \
  --report-format html
```

---

## 14. References

### Normative References

- **[RFC 2119]** Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997. https://www.ietf.org/rfc/rfc2119.txt

### Informative References

- GitHub Issues and Pull Requests documentation.
- GitHub Actions workflow run and artifact documentation.
- **[CVSS]** FIRST, "Common Vulnerability Scoring System v3.1: Specification Document." https://www.first.org/cvss/v3.1/specification-document
- **[DORA]** Google Cloud DORA, "DORA's software delivery performance metrics." https://dora.dev/guides/dora-metrics-four-keys/
- **[OCTOVERSE]** GitHub, "Octoverse." https://github.blog/news-insights/octoverse/
- **[RICE]** Intercom, "RICE: Simple prioritization for product managers." https://www.intercom.com/blog/rice-simple-prioritization-for-product-managers/
- **[SPACE]** Forsgren, N., Storey, M.-A., Maddila, C., Zimmermann, T., Houck, B., and Butler, J., "The SPACE of Developer Productivity," ACM Queue, 2021. https://queue.acm.org/detail.cfm?id=3454124

---

## 15. Change Log

### Version 0.1.0 (Working Draft) — 2026-06-25

- **Added**: Initial Impact Score feature specification.
- **Added**: Core model, attribution model, artifact, dashboard, security, and compliance requirements.
- **Added**: Informative rationale describing the workflow portfolio management origins, GitHub-native work graph design, and value/cost pairing motivation.
- **Added**: Required compliance tests T-IS-001 through T-IS-012.
- **Added**: Structured score explanation fields for policy path, version, digest, and matched rules.
- **Changed**: Clarified that the Impact Runner combines GitHub API work graph data, `gh aw logs --json` cost observations, and local normalized cache artifacts.
- **Changed**: Defined omitted output paths as unique temporary directories, with explicit `--out` reserved for durable cache artifacts.
- **Changed**: Defined `.github/workflows/aw.json` `impact.rules` as the safe default repository scoring policy format, with a typed internal representation for evaluation.
- **Changed**: Replaced highest-only dashboard scoring rails with filterable scored work item tables and removed the top workflow summary metric.
- **Changed**: Replaced the standalone `cmd/impact-score` surface with a hidden `gh aw impact` command backed by `pkg/impactscore/runner`.