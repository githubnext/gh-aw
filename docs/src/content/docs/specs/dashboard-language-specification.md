---
title: Dashboard Language Specification
description: A declarative YAML language for agentic workflow dashboards.
sidebar:
  order: 1362
---

# Dashboard Language Specification

**Version:** 0.1.0
**Status:** Working Draft
**Editor:** GitHub Agentic Workflows Team

---

## Abstract

This specification defines a small, declarative, YAML-based language for describing dashboards about organizations, repositories, agentic workflows, runs, experiments, graders, evals, usage, findings, and operational value. A dashboard contains built-in pages or custom pages. Custom pages use a constrained Vega-inspired model composed of `source`, optional `data`, `mark`, and `encoding`. This specification defines intrinsic domain semantics, aggregation and filtering rules, provenance and freshness requirements, explicit unavailable-data states, links, conformance, and compliance tests. It does not define data retrieval, implementation architecture, or rendering technology.

## Status of This Document

This document is a Working Draft and may be updated, replaced, or made obsolete. It is intended for review and implementation feedback and is not a final recommendation.

The GitHub Agentic Workflows Team maintains this document. Version numbers follow Semantic Versioning. Working Draft publication does not imply endorsement by any standards body.

Sections containing numbered requirements are normative. Examples, notes, rationales, and appendices identified as informative are non-normative unless stated otherwise.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Terminology and Conceptual Model](#3-terminology-and-conceptual-model)
4. [YAML Document Model](#4-yaml-document-model)
5. [Intrinsic Semantic Model](#5-intrinsic-semantic-model)
6. [Scope, Time, and Filters](#6-scope-time-and-filters)
7. [Dimensions, Measures, and Aggregation](#7-dimensions-measures-and-aggregation)
8. [Provenance, Freshness, and Data States](#8-provenance-freshness-and-data-states)
9. [Links and Findings](#9-links-and-findings)
10. [Built-in Pages](#10-built-in-pages)
11. [Custom Pages](#11-custom-pages)
12. [Validation and Errors](#12-validation-and-errors)
13. [Security, Privacy, and Accessibility](#13-security-privacy-and-accessibility)
14. [Compliance Testing](#14-compliance-testing)
15. [References](#15-references)
16. [Change Log](#16-change-log)
17. [Appendices](#appendices)

---

## 1. Introduction

### 1.1 Purpose

The Dashboard Language provides a portable vocabulary for defining what an agentic-operations dashboard communicates without prescribing how data is fetched, stored, cached, routed, or rendered.

### 1.2 Scope

This specification covers:

- a single-document YAML format;
- built-in and custom dashboard pages;
- intrinsic agentic-operations entities and observations;
- dimensions, measures, aggregation, scope, time, and filters;
- provenance, freshness, missing-data semantics, and links; and
- validation, conformance, and compliance testing.

This specification does not cover:

- arbitrary scripts, joins, formulas, expressions, or templates;
- plugins, themes, renderer details, or implementation architecture;
- deployment, routing, fetching, authentication, caching, or storage;
- campaign or experiment management; or
- causal inference.

### 1.3 Design Goals

The language is designed to be minimal, deterministic, auditable, and safe to validate. Built-in pages provide useful defaults. Custom pages provide only metric, table, chart, and time-series views.

### 1.4 Basis and Domain Additions

The built-in page requirements are grounded in reviewed Central Agentic Ops surfaces: an overview with rollout-mode filtering, workflow inventory and active state, run status and conclusion trends and counts, repository and workflow rankings, largest AIC spenders, findings linked to issues, pull requests, or runs, operational-value timelines, explicit provenance and freshness, and empty or unavailable states.

Engine, requested-model, and resolved-model dimensions are GitHub Agentic Workflows domain requirements. They are not represented here as observed Central Agentic Ops surface behavior.

---

## 2. Conformance

### 2.1 Requirements Notation

> The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.2 Conformance Classes

This specification defines three conformance classes:

1. **Dashboard document:** one YAML document claiming this language version.
2. **Validator:** parses a dashboard document and reports validity.
3. **Presenter:** consumes a valid document and conforming logical data to expose the specified information.

### 2.3 Normative Conformance Requirements

- **DLS-CONF-001:** A conformance claim **MUST** identify its class, specification version, implementation version when applicable, and test-suite result.
- **DLS-CONF-002:** A conforming dashboard document **MUST** satisfy all `DLS-DOC-*` requirements.
- **DLS-CONF-003:** A conforming validator **MUST** enforce all `DLS-DOC-*`, `DLS-VAL-*`, and parser-applicable `DLS-SAFE-*` requirements.
- **DLS-CONF-004:** A conforming presenter **MUST** satisfy all `DLS-SEM-*`, `DLS-CTX-*`, `DLS-AGG-*`, `DLS-DATA-*`, `DLS-LINK-*`, `DLS-PAGE-*`, `DLS-VIEW-*`, presenter-applicable `DLS-SAFE-*`, and `DLS-TEST-*` requirements.
- **DLS-CONF-005:** A non-conforming implementation **MAY** document supported subsets but **MUST NOT** claim conformance to this specification.

---

## 3. Terminology and Conceptual Model

### 3.1 Terms

| Term | Meaning |
|---|---|
| Dashboard | One named collection of ordered pages and shared defaults. |
| Built-in page | A page whose semantic content is defined by Section 10. |
| Custom page | A page containing one or more declarative views. |
| Dimension | A categorical, identifying, or temporal value used to group or filter observations. |
| Measure | A numeric observation that may be aggregated only according to its declared semantics. |
| Observation | A recorded value with time, provenance, and data-quality metadata. |
| Raw token | A provider-reported token count in one token class; not a currency or normalized cost. |
| AI Credits (`aic`) | A normalized usage or accounting measure supplied by an authoritative source; not a token count. |
| Run conclusion | The terminal GitHub Actions result of a completed run. |
| Outcome | A later repository-state evaluation of a safe output, distinct from run status and conclusion. |
| Operational value | Absolute operational attainment in `[0,1]` or `null`; not evidence of causation. |

### 3.2 Entity Relationships

```text
organization
  └─ repository
       └─ workflow
            └─ run
                ├─ experiment assignment
                ├─ usage observations
                ├─ grader observations
                ├─ eval observations
                ├─ outcome observations
                ├─ findings
                └─ operational-value observations
```

Graders and evals are definitions. Grader observations and eval observations are records produced using those definitions. An experiment assignment associates one run with one named variant, but this language does not manage experiments.

### 3.3 Normative Semantic Foundations

- **DLS-SEM-001:** An implementation **MUST** model an organization as the parent scope of zero or more repositories.
- **DLS-SEM-002:** An implementation **MUST** model a repository as belonging to exactly one organization and as containing zero or more workflows.
- **DLS-SEM-003:** An implementation **MUST** model a workflow as belonging to exactly one repository and a run as an execution of exactly one workflow.
- **DLS-SEM-004:** Workflow active state **MUST** use `true`, `false`, or `unknown`; `unknown` **MUST NOT** be treated as either Boolean value.
- **DLS-SEM-005:** Run lifecycle status **MUST** use `queued`, `in-progress`, `completed`, or `unknown`; upstream `in_progress` **MUST** normalize to `in-progress`.
- **DLS-SEM-006:** Run conclusion **MUST** use `success`, `failure`, `cancelled`, `timed-out`, `action-required`, `neutral`, `skipped`, `stale`, `startup-failure`, or `unknown`; upstream underscore-separated values **MUST** normalize to kebab-case. A non-completed run **MUST** have conclusion `unknown`.
- **DLS-SEM-007:** An experiment assignment **MUST** identify an experiment, variant, and run; absence of an assignment **MUST NOT** imply membership in a control or treatment group.
- **DLS-SEM-008:** A grader observation **MUST** identify its grader, observed subject, observation time, `value`, and `status`; status **MUST** use `pass`, `fail`, `error`, or `unavailable`. It **MUST NOT** be represented as an eval observation.
- **DLS-SEM-009:** An eval observation **MUST** identify its eval, observed subject, observation time, and BinEval result of `YES`, `NO`, or `UNKNOWN`; it **MUST NOT** be represented as a grader observation.
- **DLS-SEM-010:** A usage observation **MUST** retain raw `input-tokens`, `output-tokens`, `cache-read-tokens`, `cache-write-tokens`, and `reasoning-tokens` as separate measures and **MUST NOT** label any of them as `aic`.
- **DLS-SEM-011:** An AIC observation **MUST** be represented by `aic` and **MUST NOT** be inferred from raw tokens unless the data provenance identifies an authoritative conversion.
- **DLS-SEM-012:** A run-associated usage observation **MUST** preserve `engine`, `requested-model`, and `resolved-model` as distinct dimensions; an unavailable value **MUST** be `unknown`.
- **DLS-SEM-013:** An operational-value observation **MUST** identify its definition, operational case, evaluator digest, observed subject, observation time, requested evidence time, effective evidence cutoff, maturity time and status, and accepted evidence provenance. Its primary value **MUST** be absolute attainment in `[0,1]` or `null`.
- **DLS-SEM-014:** An implementation **MUST NOT** present experiment, grader, eval, usage, outcome, finding, or operational-value associations as causal conclusions.
- **DLS-SEM-015:** An outcome observation **MUST** identify its safe output and use `accepted`, `rejected`, `ignored`, `pending`, `lifecycle`, or `lifecycle-close`; upstream `lifecycle_close` **MUST** normalize to `lifecycle-close`. It **MUST NOT** be represented as a run conclusion.
- **DLS-SEM-016:** An optional operational-value baseline delta **MUST** remain separate from the primary absolute value and **MUST NOT** replace it.

---

## 4. YAML Document Model

### 4.1 Root Structure

The media type is not assigned by this specification. Files conventionally use `.yaml` or `.yml`.

```yaml
language-version: "0.1.0"
dashboard:
  id: example-dashboard
  title: Example Dashboard
  defaults:
    scope: {}
    time: {}
    filters: {}
  pages: []
```

### 4.2 Vocabulary

Language keys and enumerated values use canonical kebab-case. Human-readable titles and descriptions are unrestricted Unicode strings. Domain identifiers such as `owner/repository` and workflow paths retain their domain syntax.

| Mapping | Allowed keys |
|---|---|
| Root | `language-version`, `dashboard` |
| `dashboard` | `id`, `title`, `description`, `defaults`, `pages` |
| `defaults` | `scope`, `time`, `filters` |
| Built-in page | `id`, `kind`, `page`, `title`, `description` |
| Custom page | `id`, `kind`, `title`, `description`, `views` |
| View | `id`, `title`, `description`, `data`, `mark`, `encoding` |
| View `data` | `source`, `scope`, `time`, `filters`, `limit`, `order-by` |
| Field definition | `field`, `type`, `aggregate`, `time-unit`, `title` |

### 4.3 Normative Document Requirements

- **DLS-DOC-001:** A dashboard file **MUST** be valid YAML 1.2 and contain exactly one YAML document whose root is a mapping.
- **DLS-DOC-002:** The root **MUST** contain exactly `language-version` and `dashboard`.
- **DLS-DOC-003:** `language-version` **MUST** be the quoted string `"0.1.0"`.
- **DLS-DOC-004:** `dashboard` **MUST** contain a non-empty `id`, non-empty `title`, and non-empty `pages` sequence.
- **DLS-DOC-005:** Dashboard, page, and view IDs **MUST** match `^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$` and page IDs and view IDs **MUST** each be unique within their containing sequence.
- **DLS-DOC-006:** Language keys and enumerated values defined by this specification **MUST** use their exact canonical kebab-case spelling.
- **DLS-DOC-007:** A validator **MUST** reject unknown keys, unknown enumerated values, and duplicate mapping keys.
- **DLS-DOC-008:** `defaults`, when present, **MUST** be a mapping containing only `scope`, `time`, and `filters`.
- **DLS-DOC-009:** Every page **MUST** set `kind` to `built-in` or `custom` and satisfy the corresponding page shape in Sections 10 or 11.
- **DLS-DOC-010:** Titles and descriptions **MUST** be strings; IDs, references, and timestamps **MUST NOT** rely on YAML implicit type coercion.

---

## 5. Intrinsic Semantic Model

### 5.1 Logical Sources and Grain

The `source` vocabulary is closed in version 0.1.0.

| Source | One row represents | Core fields |
|---|---|---|
| `organizations` | organization | `organization`, `organization-name`, `observed-at` |
| `repositories` | repository | `organization`, `repository`, `repository-name`, `rollout-mode`, `observed-at` |
| `workflows` | workflow | `organization`, `repository`, `workflow`, `workflow-name`, `workflow-active`, `rollout-mode`, `observed-at` |
| `runs` | run | `organization`, `repository`, `workflow`, `run`, `started-at`, `ended-at`, `run-status`, `run-conclusion`, `rollout-mode`, `engine`, `requested-model`, `resolved-model` |
| `experiments` | experiment | `experiment`, `experiment-name`, `observed-at` |
| `experiment-assignments` | experiment assignment | scope IDs, `run`, `experiment`, `variant`, `observed-at` |
| `graders` | grader definition | `grader`, `grader-name`, `observed-at` |
| `grader-observations` | grader observation | scope IDs, `run`, `experiment`, `grader`, `value`, `status`, `rollout-mode`, `observed-at` |
| `evals` | eval definition | `eval`, `eval-name`, `eval-question`, `requested-model`, `observed-at` |
| `eval-observations` | eval observation | scope IDs, `run`, `experiment`, `eval`, `eval-result`, `requested-model`, `resolved-model`, `rollout-mode`, `observed-at` |
| `usage` | model invocation | scope IDs, `run`, `invocation`, `engine`, `requested-model`, `resolved-model`, `rollout-mode`, `input-tokens`, `output-tokens`, `cache-read-tokens`, `cache-write-tokens`, `reasoning-tokens`, `aic`, `observed-at` |
| `outcomes` | safe-output outcome observation | scope IDs, `run`, `safe-output`, `outcome-state`, `evidence-strength`, `observed-at`, `issue-link`, `pull-request-link`, `run-link`, `external-link` |
| `findings` | finding | scope IDs, `run`, `finding`, `finding-severity`, `finding-status`, `finding-summary`, `observed-at`, `issue-link`, `pull-request-link`, `run-link`, `external-link` |
| `operational-values` | value observation | scope IDs, `run`, `experiment`, `operational-case`, `evaluator-digest`, `rollout-mode`, `operational-value`, `operational-value-definition`, `requested-evidence-at`, `evidence-cutoff`, `maturity-at`, `maturity-status`, `delta-from-baseline`, `observed-at`, `evidence-link` |

“Scope IDs” means the applicable `organization`, `repository`, and `workflow` fields. Fields that do not apply to an observation are absent rather than fabricated. Link-bearing source fields are relation-specific optional fields whose intrinsic type is one Section 9.1 link object. `issue-link`, `pull-request-link`, `run-link`, `evidence-link`, and `external-link` correspond to the `issue`, `pull-request`, `run`, `evidence`, and `external` link relations, respectively; a source row MUST NOT encode multiple link relations inside one field.

### 5.2 Raw Token Classes

The canonical raw-token measures are `input-tokens`, `output-tokens`, `cache-read-tokens`, `cache-write-tokens`, and `reasoning-tokens`. They remain separate because provider reporting conventions may overlap.

### 5.3 Graders, Evals, and Operational Value

A grader applies a named grading criterion and produces a deterministic grader observation. An eval is a binary evaluation question and produces a `yes`, `no`, or `unknown` observation; it may use an AI model. Operational value is a separate absolute-attainment observation with an evidence cutoff and maturity time. These concepts are not interchangeable.

### 5.4 Normative Source Requirements

- **DLS-SEM-017:** A custom view `data.source` **MUST** name exactly one source from Section 5.1.
- **DLS-SEM-018:** Each logical source **MUST** preserve the grain declared in Section 5.1; duplicated observations **MUST** retain distinct observation identifiers in provenance.
- **DLS-SEM-019:** A `usage` row **MUST** represent one model invocation and **MUST NOT** repeat invocation-level AIC across token-class rows.
- **DLS-SEM-020:** Grader values, eval results, AIC, each raw-token measure, outcome states, and operational value **MUST** remain separately named throughout filtering, aggregation, and presentation.
- **DLS-SEM-021:** `rollout-mode` **MUST** use `staged`, `review`, `live`, or `unknown`.

---

## 6. Scope, Time, and Filters

### 6.1 Scope

`scope` is a mapping whose allowed keys are `organizations`, `repositories`, and `workflows`. Each value is a non-empty sequence of domain identifiers. A missing key is unbounded at that scope level.

### 6.2 Time

`time` is a mapping containing either `range` or optional `start` and `end` RFC 3339 timestamps. `range` is a positive integer followed by `h`, `d`, or `w`, such as `30d`. A relative range resolves to `[evaluated-at - range, evaluated-at)`, where the presenter exposes one RFC 3339 `evaluated-at` instant for the dashboard. Absolute `start` is inclusive and `end` is exclusive. Missing absolute bounds are unbounded. Time comparisons use instants; calendar time units use UTC.

### 6.3 Filters

`filters` maps a canonical dimension to either one scalar value or a non-empty sequence of values. Values within a sequence are alternatives; separate filter keys are conjunctive. `rollout-mode` is an ordinary dimension and follows the same rules as every other filterable dimension.

### 6.4 Context Composition

Dashboard defaults establish the initial context. A custom view's `data` context narrows that context. It cannot expand it.

- **DLS-CTX-001:** Scope constraints at different levels **MUST** be combined by intersection and **MUST** preserve organization–repository–workflow ancestry.
- **DLS-CTX-002:** `time.start` and `time.end` **MUST** be RFC 3339 timestamps, and `start` **MUST** precede `end`.
- **DLS-CTX-003:** Time filtering **MUST** include observations at `start` and exclude observations at `end`.
- **DLS-CTX-004:** A scalar filter **MUST** use equality; sequence values **MUST** use logical OR; distinct filter keys **MUST** use logical AND.
- **DLS-CTX-005:** A view context **MUST** inherit omitted dashboard defaults and **MUST** combine supplied scope, time, and filters by intersection.
- **DLS-CTX-006:** `rollout-mode` **MUST** be filterable, groupable, and displayable by the same mechanisms as other dimensions.
- **DLS-CTX-007:** Missing or `unknown` dimension values **MUST NOT** match a concrete filter value and **MUST** match the explicit value `unknown`.
- **DLS-CTX-008:** Filtering **MUST** occur before aggregation, ordering, and limiting.
- **DLS-CTX-009:** `time.range` **MUST** match `^[1-9][0-9]*(h|d|w)$` and **MUST NOT** appear with `start` or `end`.
- **DLS-CTX-010:** A presenter resolving `time.range` **MUST** expose `evaluated-at` and use it consistently for every page and view in the dashboard.

---

## 7. Dimensions, Measures, and Aggregation

### 7.1 Canonical Dimensions

Canonical dimensions include entity IDs, `variant`, `workflow-active`, `run-status`, `run-conclusion`, `outcome-state`, `rollout-mode`, `engine`, `requested-model`, `resolved-model`, operational-value definition, categorical observation results, and temporal fields.

### 7.2 Canonical Measures

| Measure | Meaning | Additivity |
|---|---|---|
| `input-tokens` | Provider-reported input tokens | Additive |
| `output-tokens` | Provider-reported output tokens | Additive |
| `cache-read-tokens` | Provider-reported cache-read tokens | Additive |
| `cache-write-tokens` | Provider-reported cache-write tokens | Additive |
| `reasoning-tokens` | Provider-reported reasoning tokens | Additive |
| `aic` | Authoritatively supplied AI Credits | Additive |
| `value` on `grader-observations` | Value emitted by a grader | Non-additive by default |
| `operational-value` | Absolute attainment under a named definition | Non-additive by default |

Entity counts are obtained with `count` or `distinct-count`; they are not stored measures.

### 7.3 Aggregates

Allowed aggregate values are `count`, `distinct-count`, `sum`, `mean`, `min`, `max`, and `none`. Omitted `aggregate` means `none`. `count` counts non-null field values. `distinct-count` counts distinct non-null values.

Allowed `time-unit` values are `hour`, `day`, `week`, and `month`. Buckets are half-open UTC intervals. Weeks begin Monday at 00:00:00Z; months begin on the first day.

Unaggregated dimensions in an encoding form the grouping key. Aggregated fields are computed once per resulting group. A metric with no unaggregated dimension computes one value over its effective context.

### 7.4 Normative Aggregation Requirements

- **DLS-AGG-001:** An implementation **MUST** group only by dimensions and **MUST** aggregate only measures or entity identifiers compatible with the selected aggregate.
- **DLS-AGG-002:** `sum` **MUST** be accepted only for the five raw-token measures and `aic`.
- **DLS-AGG-003:** Different raw-token measures **MUST NOT** be combined into a derived total because provider reporting classes may overlap; a combined presentation **MUST** retain separate measures.
- **DLS-AGG-004:** AIC aggregation **MUST** sum only available, non-negative AIC observations, retain all contributing provenance, and **MUST NOT** substitute zero for missing AIC.
- **DLS-AGG-005:** Grader `value` and `operational-value` **MUST** use `none`, `mean`, `min`, or `max`, and aggregation **MUST** retain grader identity or operational-value definition, respectively.
- **DLS-AGG-006:** `count` and `distinct-count` **MUST** ignore absent values and **MUST NOT** substitute zero.
- **DLS-AGG-007:** A time unit **MUST** be applied before grouping and **MUST** use the UTC boundaries in Section 7.3.
- **DLS-AGG-008:** Rankings **MUST** disclose the ranked measure, direction, filters, time range, scope, and tie behavior; ties **MUST** then be ordered by canonical entity ID ascending.

---

## 8. Provenance, Freshness, and Data States

### 8.1 Data-Set Metadata

Each logical source is accompanied by metadata outside the dashboard YAML:

| Field | Meaning |
|---|---|
| `source-id` | Stable identifier for the supplying source |
| `source-kind` | Human-readable source category |
| `as-of` | Latest observation time represented |
| `retrieved-at` | Time the data was made available to the presenter |
| `coverage-start`, `coverage-end` | Known half-open coverage interval |
| `completeness` | `complete`, `partial`, or `unknown` |
| `freshness` | `fresh`, `stale`, or `unknown` |
| `provenance-link` | Optional safe link to source evidence |

Freshness is an asserted data property. This specification does not define a cache or infer a universal staleness threshold.

### 8.2 Data States

Data quality has three independent axes:

| Axis | Values |
|---|---|
| Availability | `available`, `empty`, `unavailable` |
| Completeness | `complete`, `partial`, `unknown` |
| Freshness | `fresh`, `stale`, `unknown` |

`empty` means a valid selection returned no observations. It may be partial, stale, or unknown on the other axes. `unavailable` means no usable result exists.

### 8.3 Normative Data Requirements

- **DLS-DATA-001:** Every consumed logical source **MUST** provide `source-id`, `source-kind`, `as-of`, `retrieved-at`, `completeness`, and `freshness`.
- **DLS-DATA-002:** Provenance and freshness **MUST** remain associated with derived metrics, tables, charts, rankings, and links.
- **DLS-DATA-003:** A presenter **MUST** expose `as-of`, freshness, completeness, and source identity for every page or view.
- **DLS-DATA-004:** An empty selection **MUST** have availability `empty`; `count` and `distinct-count` over that selection **MUST** produce zero, while other aggregates **MUST** remain absent.
- **DLS-DATA-005:** An unavailable result **MUST** identify the affected source and **MUST NOT** fabricate observations or carry forward an unmarked previous value.
- **DLS-DATA-006:** A partial result **MUST** identify known missing scope or time coverage and **MUST NOT** be labeled complete.
- **DLS-DATA-007:** A stale result **MUST** retain its original `as-of` value and **MUST** be explicitly identified as stale.
- **DLS-DATA-008:** Availability, completeness, and freshness **MUST** remain separate; `unknown` completeness or freshness **MUST** remain distinct from every known value on the same axis.

---

## 9. Links and Findings

### 9.1 Link Model

A link has `relation`, `href`, and `label`. Allowed relations are `organization`, `repository`, `workflow`, `run`, `issue`, `pull-request`, `evidence`, and `external`. When a relation-specific link field from Section 5.1 is present on a source row, it contains exactly one link object whose `relation` matches the field name.

A finding is an observation with a stable finding ID, summary, status, severity, observation time, provenance, applicable scope, and zero or more relation-specific link fields. Finding status uses `open`, `resolved`, `dismissed`, or `unknown`. Severity uses `critical`, `high`, `medium`, `low`, `informational`, or `unknown`.

### 9.2 Normative Link Requirements

- **DLS-LINK-001:** Every link **MUST** contain one allowed `relation`, an absolute HTTPS `href`, and a non-empty `label`.
- **DLS-LINK-002:** Links **MUST** retain the provenance and subject association from which they were derived.
- **DLS-LINK-003:** A finding or outcome **MUST** expose relation-specific links to its associated issue, pull request, or run when those associations are available.
- **DLS-LINK-004:** A finding, outcome, or operational-value observation without an available link association **MUST** remain valid and **MUST NOT** contain a fabricated link.
- **DLS-LINK-005:** A relation-specific link field, when present, **MUST** contain exactly one Section 9.1 link object and **MUST NOT** contain a sequence, mapping of multiple relations, or scalar URL.

---

## 10. Built-in Pages

### 10.1 Syntax

```yaml
- id: runs
  kind: built-in
  page: runs
  title: Runs
```

Allowed built-in page names are:

`overview`, `organizations`, `repositories`, `workflows`, `runs`, `experiments`, `graders`, `evals`, `usage`, `engines-models`, `operational-value`, and `findings`.

### 10.2 Required Content

- **DLS-PAGE-001:** A built-in page **MUST** contain `id`, `kind: built-in`, and one allowed `page`; an omitted title **MUST** default to the page name with words capitalized.
- **DLS-PAGE-002:** The `overview` page **MUST** expose rollout-mode filtering, workflow active-state inventory, run status and conclusion counts and trends, repository and workflow rankings, largest AIC spenders, recent linked findings, an operational-value timeline, and provenance and freshness.
- **DLS-PAGE-003:** The `organizations` page **MUST** expose organization inventory, repository count, workflow count, run count, available usage measures, and data state by organization.
- **DLS-PAGE-004:** The `repositories` page **MUST** expose repository inventory and rankings by run count, AIC, and available operational value without combining different operational-value definitions.
- **DLS-PAGE-005:** The `workflows` page **MUST** expose workflow inventory, active state, rollout mode, run count, run conclusions, downstream outcome counts, available usage, findings, and operational value.
- **DLS-PAGE-006:** The `runs` page **MUST** expose run status trends and counts, terminal conclusions, downstream outcome observations when available, scope, rollout mode, engine, requested model, resolved model, time, and run links.
- **DLS-PAGE-007:** The `experiments` page **MUST** expose experiment definitions and observed run-to-variant assignments, grader observations, eval observations, outcomes, usage, and operational value without claiming causation.
- **DLS-PAGE-008:** The `graders` page **MUST** keep grader definitions and grader observations distinguishable and expose observed subject, result, score when present, time, and provenance.
- **DLS-PAGE-009:** The `evals` page **MUST** keep eval definitions and eval observations distinguishable and expose observed subject, `YES`, `NO`, or `UNKNOWN` result, evaluation model when available, time, and provenance.
- **DLS-PAGE-010:** The `usage` page **MUST** present each raw-token measure separately from AIC and expose engine, requested model, resolved model, scope, rollout mode, time, and provenance.
- **DLS-PAGE-011:** The `engines-models` page **MUST** expose engine, requested model, and resolved model as separate dimensions with run counts, run conclusions, downstream outcomes, raw tokens, and AIC where available.
- **DLS-PAGE-012:** The `operational-value` page **MUST** expose a time-ordered absolute-attainment series with definition, operational case, evaluator digest, subject, requested evidence time, effective evidence cutoff, maturity time and status, accepted evidence provenance, freshness, applicable experiment assignment, and separate baseline delta when available.
- **DLS-PAGE-013:** The `findings` page **MUST** expose finding summary, severity, status, scope, time, provenance, and available issue, pull-request, and run links.
- **DLS-PAGE-014:** Every built-in page **MUST** honor the dashboard scope, time, and filters and expose availability, completeness, and freshness independently.

---

## 11. Custom Pages

### 11.1 Syntax and View Classes

A custom page contains a non-empty `views` sequence. Each view has one `data` mapping, one mark, and one encoding.

| Semantic view | `mark` values | Required encoding |
|---|---|---|
| Metric | `metric` | `value` |
| Table | `table` | `columns` |
| Chart | `chart` | `x`, `y` |

Allowed encoding channels are `value`, `columns`, `x`, `y`, `color`, and `href`. `columns` is a non-empty sequence of field definitions; other channels contain one field definition. The `href` channel references a link-typed source field and does not select from multiple links.

Field `type` values are `nominal`, `ordinal`, `quantitative`, and `temporal`. When omitted, type defaults to the intrinsic field type. A field title defaults to its kebab-case field name with words capitalized.

A `chart` with temporal `x` has a line time-series default. Other charts have a bar default. These known defaults are semantic; this specification does not define visual styling.

### 11.2 Data Narrowing

View `data` contains `source` and may also contain:

- `scope`, `time`, and `filters` as defined in Section 6;
- `limit`, a positive integer; and
- `order-by`, a non-empty sequence of mappings containing `field` and `direction`, where direction is `asc` or `desc`.

An omitted `data` inherits dashboard defaults. Omitted `limit` means no language-level limit. Omitted `order-by` orders entity-grain rows by canonical entity ID ascending and leaves aggregate groups ordered by their encoded dimensions ascending.

### 11.3 Normative Custom-View Requirements

- **DLS-VIEW-001:** A custom page **MUST** contain `id`, `kind: custom`, and a non-empty `views` sequence; an omitted title **MUST** default from its page ID.
- **DLS-VIEW-002:** Every view **MUST** contain a unique `id`, a `data` mapping with one canonical `source`, one allowed `mark`, and an `encoding` mapping.
- **DLS-VIEW-003:** `metric` **MUST** encode exactly one `value` field and **MAY** encode `href`; it **MUST NOT** encode chart or table channels.
- **DLS-VIEW-004:** `table` **MUST** encode non-empty `columns` and **MAY** encode `href`; it **MUST NOT** encode `value`, `x`, `y`, or `color`.
- **DLS-VIEW-005:** `chart` **MUST** encode `x` and quantitative `y`, **MAY** encode `color` and `href`, and **MUST NOT** encode `value` or `columns`.
- **DLS-VIEW-006:** A `chart` with temporal `x` **MUST** use the line time-series default; any other valid `chart` **MUST** use the bar default.
- **DLS-VIEW-007:** An encoding field **MUST** exist in the selected source and its declared type **MUST** be compatible with its intrinsic type or aggregate output type; an `href` field **MUST** have intrinsic type link.
- **DLS-VIEW-008:** A field definition **MUST** contain `field` and **MAY** contain only `type`, `aggregate`, `time-unit`, and `title` in addition.
- **DLS-VIEW-009:** `time-unit` **MUST** be used only with a temporal field and **MUST** use an allowed value from Section 7.3.
- **DLS-VIEW-010:** `data.limit` **MUST** be a positive integer, and `data.order-by` fields **MUST** exist in the selected source or be encoded aggregate outputs.
- **DLS-VIEW-011:** A custom view **MUST NOT** contain scripts, joins, formulas, expressions, templates, plugins, or undeclared transforms.
- **DLS-VIEW-012:** A custom view **MUST** apply defaults, filtering, aggregation, ordering, and limiting in the order defined by Sections 6, 7, and 11.2.
- **DLS-VIEW-013:** A custom view **MUST** expose its source provenance, freshness, completeness, effective scope, effective time range, and effective filters.
- **DLS-VIEW-014:** A presenter rendering `href` **MUST** use the referenced link object's `href` as the navigation target and **MUST** expose the link object's `label` as the accessible link label. If the referenced link field is absent for a datum, the datum **MUST** remain valid and **MUST** render without a link.

---

## 12. Validation and Errors

Validation proceeds conceptually through YAML syntax, document count, structural vocabulary, references and types, semantic compatibility, and safety constraints. This order does not prescribe implementation architecture.

- **DLS-VAL-001:** A validator **MUST** report every detected error with an error code, a human-readable message, and a location identifying the nearest YAML path.
- **DLS-VAL-002:** A validator **MUST** reject a document when any Level 1 structural requirement fails.
- **DLS-VAL-003:** A Level 2 or Level 3 validator **MUST** reject incompatible source fields, filters, aggregates, encodings, links, or data relationships. A validator **MUST** reject an `href` reference to a non-link field or an ambiguous multi-link field with link-specific error code `DLS-E009`.
- **DLS-VAL-004:** Error reporting **MUST NOT** expose credentials, secret values, or sensitive source payloads.

Error codes are listed in Appendix B.

---

## 13. Security, Privacy, and Accessibility

### 13.1 Security

Dashboard documents are declarative data, not executable programs. Untrusted YAML, labels, links, and provenance values may be attacker-controlled.

### 13.2 Privacy

Run, finding, grader, eval, usage, and provenance data may identify people, repositories, or confidential work. Data minimization and access control occur outside this language, but presentations need to preserve data-quality and provenance truthfully.

### 13.3 Accessibility

Accessible semantics apply independently of visual renderer choice. Each view has a title, table fields have labels, links have labels, and data states have text equivalents.

### 13.4 Normative Safety Requirements

- **DLS-SAFE-001:** A parser **MUST** use YAML safe-loading behavior and **MUST** reject custom tags, aliases, and cyclic structures.
- **DLS-SAFE-002:** A processor **MUST NOT** execute document content or interpret any field as code, a template, a command, or a network-fetch instruction.
- **DLS-SAFE-003:** Human-readable document and data strings **MUST** be treated as text and **MUST NOT** be interpreted as markup without context-appropriate sanitization.
- **DLS-SAFE-004:** Link handling **MUST** reject credentials in URIs and every scheme other than `https`.
- **DLS-SAFE-005:** Documents and provenance **MUST NOT** contain authentication credentials, secret tokens, or private keys.
- **DLS-SAFE-006:** A presenter **MUST** expose only observations and links permitted by the consuming context and **MUST NOT** imply that language validity grants data access.
- **DLS-SAFE-007:** Every page and view **MUST** have a non-empty accessible name, using its title or title default.
- **DLS-SAFE-008:** Metrics, charts, and time series **MUST** expose a textual value or tabular equivalent, and tables **MUST** expose labeled columns.
- **DLS-SAFE-009:** Color **MUST NOT** be the only means of communicating a category, status, outcome, freshness, completeness, or severity.
- **DLS-SAFE-010:** Every availability, completeness, and freshness value **MUST** have a distinct textual label, and each link **MUST** expose its non-empty label.

---

## 14. Compliance Testing

### 14.1 Test Suite Requirements

A compliance suite uses valid and invalid YAML fixtures, logical data fixtures with explicit source metadata, and deterministic expected semantic outputs. Tests do not require a particular renderer.

- **DLS-TEST-001:** A conformance test suite **MUST** exercise every normative requirement applicable to the claimed class and level.
- **DLS-TEST-002:** Each test result **MUST** record test ID, requirement ID, implementation version, pass or fail status, and failure evidence.
- **DLS-TEST-003:** Tests involving time **MUST** include exact start and end boundaries; tests involving missing data **MUST** distinguish absent, zero, empty, unavailable, partial, stale, and unknown.

### 14.2 Compliance Checklist

In the table, “accept” means validation succeeds; “reject” means validation fails with an applicable error; “expose” means the semantic output contains the listed information.

| Requirement | Test ID | Level | Procedure and expected outcome |
|---|---|---:|---|
| DLS-CONF-001–005 | T-CONF-001 | 1–3 | Inspect full and partial claims; verify labels, coverage, results, and enumerated gaps. |
| DLS-DOC-001–010 | T-DOC-001 | 1 | Apply positive and negative syntax, root, version, identity, vocabulary, defaults, page-shape, and scalar-type fixtures. |
| DLS-SEM-001–007 | T-SEM-001 | 2 | Validate entity ancestry, active state, run status, run conclusion, and explicit experiment assignments. |
| DLS-SEM-008–016 | T-SEM-002 | 2 | Distinguish grader, eval, tokens, AIC, run conclusions, outcomes, engine/models, and value; reject causal labeling. |
| DLS-SEM-017–021 | T-SEM-003 | 2 | Validate source vocabulary, grain, token classes, rollout modes, and distinct measure names. |
| DLS-CTX-001–008 | T-CTX-001 | 2 | Exercise ancestry, boundary times, Boolean filter rules, inheritance, rollout mode, unknown, and operation order. |
| DLS-AGG-001–008 | T-AGG-001 | 2 | Exercise allowed aggregates, compatibility, nulls, UTC buckets, ranking disclosure, and deterministic ties. |
| DLS-DATA-001–008 | T-DATA-001 | 2 | Exercise required metadata, derivation traceability, and each distinct data state. |
| DLS-LINK-001–005 | T-LINK-001 | 2 | Validate link shape, safety, provenance, available associations, absent associations, and one-link-per-field cardinality. |
| DLS-PAGE-001–014 | T-PAGE-001 | 3 | Evaluate each built-in fixture for required content, defaults, context, and data states. |
| DLS-VIEW-001–006 | T-VIEW-001 | 3 | Validate custom structure and every allowed mark/channel combination. |
| DLS-VIEW-007–014 | T-VIEW-002 | 3 | Validate fields, types, link-compatible `href`, time units, ordering, exclusions, operation order, exposed context, and link labels. |
| DLS-VAL-001–004 | T-VAL-001 | 1–3 | Verify rejection, coded path-specific errors, semantic checks, and secret redaction. |
| DLS-SAFE-001–006 | T-SAFE-001 | 3 | Exercise safe YAML, inert content, sanitization, HTTPS links, secrets, and authorization boundaries. |
| DLS-SAFE-007–010 | T-SAFE-002 | 3 | Inspect names, textual alternatives, labels, and non-color semantics. |
| DLS-TEST-001–003 | T-TEST-001 | 1–3 | Inspect coverage, result metadata, time boundaries, and missing-data distinctions. |


### 14.3 Custom Link Fixture Requirements

A Level 3 compliance suite MUST include at least one positive and one negative custom-view fixture for `href` link rendering and validation. The positive fixture MUST include a custom view whose `href.field` references a relation-specific link field and logical data containing one row where that field is present and one row where it is absent. The expected semantic output MUST use the present link object's `href` as the navigation target, expose its `label` as the accessible link label, and leave the absent-link row unlinked.

```yaml
language-version: "0.1.0"
dashboard:
  id: findings-links
  title: Findings Links
  pages:
    - id: findings-table
      kind: custom
      title: Findings with Pull Requests
      views:
        - id: open-findings
          data:
            source: findings
            filters:
              finding-status: open
          mark: table
          encoding:
            columns:
              - field: finding-summary
              - field: finding-severity
            href:
              field: pull-request-link
```

The negative fixture MUST include a custom view whose `href.field` references a field that is not link-typed, such as `finding-summary`, or an implementation extension field that contains multiple links. The expected validation result MUST reject the document with `DLS-E009`.

```yaml
language-version: "0.1.0"
dashboard:
  id: invalid-finding-links
  title: Invalid Finding Links
  pages:
    - id: findings-table
      kind: custom
      views:
        - id: invalid-href
          data:
            source: findings
          mark: table
          encoding:
            columns:
              - field: finding-summary
            href:
              field: finding-summary
```

### 14.4 Recommended Execution Procedure

1. Validate positive and negative YAML fixtures.
2. Validate semantic fixtures and relationships.
3. Evaluate context and aggregation fixtures.
4. Inspect provenance, freshness, links, and data states.
5. Evaluate every built-in page and custom mark.
6. Inspect security, privacy, and accessibility semantics.
7. Publish the conformance claim and machine-readable test results.

---

## 15. References

### 15.1 Normative References

- **[RFC 2119]** Bradner, S. *Key words for use in RFCs to Indicate Requirement Levels*. RFC 2119. <https://www.ietf.org/rfc/rfc2119.txt>
- **[RFC 3339]** Klyne, G.; Newman, C. *Date and Time on the Internet: Timestamps*. RFC 3339. <https://www.rfc-editor.org/rfc/rfc3339>
- **[RFC 3986]** Berners-Lee, T.; Fielding, R.; Masinter, L. *Uniform Resource Identifier (URI): Generic Syntax*. RFC 3986. <https://www.rfc-editor.org/rfc/rfc3986>
- **[YAML 1.2.2]** *YAML Ain't Markup Language, Version 1.2.2*. <https://yaml.org/spec/1.2.2/>
- **[AIC]** [AI Credits Specification](/gh-aw/specs/ai-credits-specification/)
- **[GRADERS]** [Graders Specification](/gh-aw/specs/graders-specification/)

### 15.2 Informative References

- **[SEMVER]** *Semantic Versioning 2.0.0*. <https://semver.org/>
- **[VEGA-LITE]** *Vega-Lite: A Grammar of Interactive Graphics*. <https://vega.github.io/vega-lite/>
- **[WCAG 2.2]** *Web Content Accessibility Guidelines (WCAG) 2.2*. W3C Recommendation. <https://www.w3.org/TR/WCAG22/>
- **[EXPERIMENTS]** [A/B Experiments Specification](/gh-aw/experimental/experiments-specification/)
- **[OUTCOMES]** [Outcomes](/gh-aw/reference/outcomes/)

---

## 16. Change Log

### Version 0.1.0 (Working Draft)

- Initial Dashboard Language specification.
- Defined intrinsic entities, observations, dimensions, measures, and relationships.
- Defined built-in pages and constrained custom views.
- Added provenance, freshness, data states, links, safety requirements, and compliance tests.

---

## Appendices

### Appendix A: Complete Example (Informative)

```yaml
language-version: "0.1.0"
dashboard:
  id: agentic-operations
  title: Agentic Operations
  description: Workflow activity, usage, findings, and operational value.
  defaults:
    scope:
      organizations:
        - octo-org
    time:
      range: 30d
    filters:
      rollout-mode:
        - review
        - live
  pages:
    - id: overview
      kind: built-in
      page: overview
    - id: workflows
      kind: built-in
      page: workflows
    - id: usage-by-repository
      kind: custom
      title: Usage by Repository
      views:
        - id: total-aic
          title: Total AI Credits
          data:
            source: usage
          mark: metric
          encoding:
            value:
              field: aic
              type: quantitative
              aggregate: sum
        - id: daily-runs
          title: Daily Runs
          data:
            source: runs
          mark: chart
          encoding:
            x:
              field: started-at
              type: temporal
              time-unit: day
            y:
              field: run
              type: quantitative
              aggregate: count
            color:
              field: rollout-mode
              type: nominal
        - id: largest-spenders
          title: Largest AIC Spenders
          data:
            source: usage
            limit: 10
            order-by:
              - field: aic
                direction: desc
          mark: table
          encoding:
            columns:
              - field: repository
                type: nominal
              - field: aic
                type: quantitative
                aggregate: sum
```

### Appendix B: Error Codes (Normative)

| Code | Meaning |
|---|---|
| `DLS-E001` | Invalid YAML syntax |
| `DLS-E002` | Invalid YAML document count or root |
| `DLS-E003` | Missing or invalid required field |
| `DLS-E004` | Unknown or duplicate key |
| `DLS-E005` | Non-canonical vocabulary or identifier |
| `DLS-E006` | Unknown source, field, or reference |
| `DLS-E007` | Incompatible mark, channel, type, or time unit |
| `DLS-E008` | Forbidden executable or transformation feature |
| `DLS-E009` | Unsafe, invalid, ambiguous, or incompatible link or `href` reference |
| `DLS-E010` | Invalid scope, filter, time range, or aggregation |
| `DLS-E011` | Invalid entity relationship or source grain |
| `DLS-E012` | Missing required provenance or data-state metadata |

### Appendix C: Invalid Examples (Informative)

#### C.1 Multiple YAML Documents

```yaml
language-version: "0.1.0"
dashboard: {}
---
language-version: "0.1.0"
dashboard: {}
```

Invalid because a file contains more than one YAML document.

#### C.2 Non-Canonical ID

```yaml
language-version: "0.1.0"
dashboard:
  id: Agentic_Operations
  title: Agentic Operations
  pages: []
```

Invalid because the ID is not kebab-case and `pages` is empty.

#### C.3 Forbidden Join and Expression

```yaml
- id: combined-usage
  kind: custom
  views:
    - id: calculated-cost
      data:
        source: usage
      join: runs
      mark: metric
      encoding:
        value:
          expression: raw-token-count * rate
```

Invalid because `join` and `expression` are not language vocabulary and arbitrary joins and expressions are excluded.

#### C.4 Incompatible Measure

```yaml
- id: summed-value
  kind: custom
  views:
    - id: value-total
      data:
        source: operational-values
      mark: metric
      encoding:
        value:
          field: operational-value
          aggregate: sum
```

Invalid because operational value is non-additive and cannot use `sum`.

### Appendix D: Semantic Distinctions (Informative)

| Concept | Example question answered | Not equivalent to |
|---|---|---|
| Raw tokens | How many provider-reported input tokens were observed? | AIC, cost, outcome, or value |
| AIC | How many authoritative AI Credits were attributed? | Raw tokens or operational value |
| Run conclusion | Did the completed run succeed, fail, time out, or end another way? | Downstream outcome, grader result, or eval result |
| Outcome | Was a safe output later accepted, rejected, pending, ignored, or otherwise classified? | Run conclusion or operational value |
| Grader observation | What result did a named grading criterion emit? | Eval observation or run conclusion |
| Eval observation | Did a named binary evaluation return `yes`, `no`, or `unknown`? | Grader observation or operational value |
| Operational value | What absolute attainment was observed under a named definition and evidence cutoff? | AIC, outcome, or causal impact |

---
