---
title: Wizard Data Model Specification
description: Formal specification for the JSON data model that drives the docs wizard (WHAT → WHEN → WHERE → task details) and is designed for reuse by a future gh aw CLI consumer.
sidebar:
  order: 1371
---

# Wizard Data Model Specification

**Version**: 1.0.0
**Status**: Draft
**Publication Date**: 2026-08-17
**Editor**: GitHub Agentic Workflows Team
**This Version**: [wizard-data-model-specification](/gh-aw/specs/wizard-data-model-specification/)
**Latest Published Version**: This document

---

## Abstract

This specification defines the JSON data model that drives the docs site wizard used to help users compose a prompt for generating a new GitHub Agentic Workflow (gh-aw). The model captures goal categories (WHAT), trigger options (WHEN), safe-output destination options (WHERE), and prompt-assembly fields, without embedding any Astro/UI-specific behavior. It is designed so that a future `gh aw` CLI consumer can load and interpret the same data without re-authoring the underlying semantics.

## Status of This Document

This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time. This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [File Location and Naming](#3-file-location-and-naming)
4. [Data Model](#4-data-model)
5. [Conditional Progression](#5-conditional-progression)
6. [Prompt Assembly](#6-prompt-assembly)
7. [Versioning and Compatibility](#7-versioning-and-compatibility)
8. [Future CLI Reuse](#8-future-cli-reuse)
9. [Examples](#9-examples)
10. [References](#references)
11. [Change Log](#change-log)

---

## 1. Introduction

### 1.1 Purpose

The docs wizard walks a user through composing a self-contained prompt that generates a new agentic workflow: first the high-level **goal** (WHAT), then the **trigger** (WHEN), then the **safe-output destination** (WHERE, mostly inferred from prior answers), and finally free-form **task details**. This specification defines the JSON shape backing that flow.

### 1.2 Scope

This specification covers:

- The top-level JSON entities: `goalCategories`, `triggerOptions`, `destinationOptions`, and `promptTemplate`.
- Required vs. optional fields and stable identifier conventions.
- How conditional progression (WHAT → WHEN → WHERE) is expressed via id references rather than embedded control flow.
- File location, naming, and versioning for compatibility with a future CLI consumer.

This specification does not define the Astro component implementation, CSS/visual presentation, or the exact wording of any generated prompt text — those are implementation details left to the docs site.

### 1.3 Terminology

The key words "MUST", "MUST NOT", "REQUIRED", "SHOULD", "SHOULD NOT", and "MAY" are to be interpreted as described in RFC 2119.

## 2. Conformance

A conforming data file:

- MUST validate against the JSON Schema at `docs/public/schemas/wizard-data-model.schema.json` (published at `/gh-aw/schemas/wizard-data-model.schema.json`).
- MUST include a `version` field using semantic versioning.
- MUST use explicit, stable `id` fields as keys for goal categories, trigger options, and destination options — free-form label text MUST NOT be used as a lookup key.
- SHOULD keep presentation metadata (`text.label`, `text.help`) minimal so behavioral/content fields remain portable across UI implementations.

## 3. File Location and Naming

| Artifact | Path | Purpose |
|---|---|---|
| JSON Schema | `docs/public/schemas/wizard-data-model.schema.json` | Normative schema, published under the docs site at `/gh-aw/schemas/wizard-data-model.schema.json`, consistent with existing schemas (e.g. `mcp-gateway-config.schema.json`). |
| Example/reference data | `docs/public/schemas/wizard-data-model.example.json` | A conforming example instance used for documentation and tests. |
| Wizard runtime data | `docs/src/data/wizard-data-model.json` (or equivalent Astro data source, introduced in a follow-up implementation issue) | The actual data consumed by the Astro wizard component at build/runtime. |

Keeping the schema under `docs/public/schemas/` follows the existing convention for gh-aw JSON schemas (see `mcp-gateway-config.schema.json`, `mcp-scripts-config.schema.json`) so it is publicly resolvable and reusable outside the docs site, including by a future CLI.

## 4. Data Model

### 4.1 Top-Level Shape

```json
{
  "version": "1.0.0",
  "goalCategories": [ /* goalCategory[] */ ],
  "triggerOptions": [ /* triggerOption[] */ ],
  "destinationOptions": [ /* destinationOption[] */ ],
  "promptTemplate": { /* promptTemplate, optional */ }
}
```

| Field | Required | Type | Description |
|---|---|---|---|
| `version` | Yes | string | Semver string for this data model instance. |
| `goalCategories` | Yes | array | WHAT-step options. Minimum one entry. |
| `triggerOptions` | Yes | array | WHEN-step options, referenced by id from `goalCategories`. |
| `destinationOptions` | Yes | array | WHERE-step options, referenced by id from `goalCategories` and inferable from trigger type. |
| `promptTemplate` | No | object | Optional fields controlling final prompt assembly. |

### 4.2 `goalCategory`

| Field | Required | Type | Description |
|---|---|---|---|
| `id` | Yes | string | Stable identifier, `^[a-z][a-z0-9-]*$`. MUST NOT change once published. |
| `text.label` | Yes | string | Short user-facing label. |
| `text.help` | No | string | Longer help text. |
| `triggerOptionIds` | Yes | string[] | IDs of compatible `triggerOptions` (min 1). |
| `destinationOptionIds` | Yes | string[] | IDs of compatible `destinationOptions` (min 1). |
| `defaultTriggerOptionId` | No | string | Pre-selected/inferred trigger id. |
| `defaultDestinationOptionId` | No | string | Pre-selected/inferred destination id. |

### 4.3 `triggerOption`

| Field | Required | Type | Description |
|---|---|---|---|
| `id` | Yes | string | Stable identifier. |
| `type` | Yes | string | One of: `schedule`, `issues`, `pull_request`, `discussion`, `issue_comment`, `workflow_dispatch`, `slash_command`, `push`, `other`. |
| `text.label` / `text.help` | Yes/No | string | Presentation metadata. |
| `frontmatter` | No | object | Literal or template fragment of gh-aw workflow frontmatter `on:` configuration contributed by this option. Free-form object; consumers merge it as-is. |

### 4.4 `destinationOption`

| Field | Required | Type | Description |
|---|---|---|---|
| `id` | Yes | string | Stable identifier. |
| `safeOutputType` | Yes | string | Name of the corresponding gh-aw `safe-outputs` handler key (e.g. `add-comment`, `create-issue`, `create-pull-request`). |
| `text.label` / `text.help` | Yes/No | string | Presentation metadata. |
| `inferFromTriggerTypes` | No | string[] | `triggerOption.type` values this destination is inferred/recommended for. |
| `frontmatter` | No | object | Literal or template fragment of gh-aw `safe-outputs:` configuration contributed by this option. |

### 4.5 `promptTemplate`

| Field | Required | Type | Description |
|---|---|---|---|
| `introText` | No | string | Fixed opening text for the generated prompt. MUST establish that the prompt is self-contained and assumes no prior gh-aw installation. |
| `sections` | No | array of `{id, heading}` | Ordered list of named sections used to assemble the final prompt (e.g. goal, trigger, destination, task-details). |

## 5. Conditional Progression

The WHAT → WHEN → WHERE flow is expressed declaratively through id references, not through embedded control-flow or Astro-specific logic:

1. The wizard renders `goalCategories` for the WHAT step.
2. Upon selection of a `goalCategory`, the WHEN step SHOULD present only the `triggerOptions` whose `id` appears in that goal's `triggerOptionIds`, pre-selecting `defaultTriggerOptionId` when present.
3. Upon selection of a `triggerOption`, the WHERE step SHOULD present only the `destinationOptions` whose `id` appears in the goal's `destinationOptionIds`. Implementations SHOULD further narrow or pre-select a destination whose `inferFromTriggerTypes` includes the selected trigger's `type`, allowing this step to be skipped or pre-answered per the parent issue's requirement that WHERE be "mostly automatically inferred."
4. The task-details step is free-form text collected by the UI and is not modeled here beyond the `task-details` prompt section id.

This mechanism keeps all conditional logic data-driven: any consumer (Astro component or CLI) walks the same three arrays using the same id-matching rules.

## 6. Prompt Assembly

Implementations assembling the final prompt MUST:

- Produce a prompt that is self-contained and runnable in the user's project without assuming gh-aw is already installed or configured.
- Include, at minimum, the selected goal's label/help, the selected trigger's resulting frontmatter intent, the selected (or inferred) destination's resulting frontmatter intent, and the free-form task details.
- Use `promptTemplate.sections` (when present) to order and label these parts; otherwise fall back to the order: goal, trigger, destination, task details.

## 7. Versioning and Compatibility

- `version` follows semver. Additive, backward-compatible changes (new optional fields, new catalog entries) increment MINOR or PATCH. Removing or renaming a required field, or changing the meaning of an existing `id`, MUST increment MAJOR.
- Consumers MUST tolerate unknown additional top-level or nested fields within the same MAJOR version (forward compatibility for future CLI/UI features).
- `id` values are permanent once published; deprecating an option is done by removing it from `goalCategory.triggerOptionIds`/`destinationOptionIds` rather than deleting the catalog entry outright, preserving compatibility for consumers caching older goal-to-option mappings.

## 8. Future CLI Reuse

A future `gh aw` CLI consumer (e.g. `gh aw new --wizard`) is expected to:

- Load the same JSON file (or the published schema URL) used by the docs site, without re-authoring goal/trigger/destination semantics.
- Interpret `frontmatter` fragments on `triggerOption` and `destinationOption` as literal gh-aw workflow frontmatter fragments to merge into a generated `.md` workflow, exactly as the docs wizard would when assembling its generated prompt text.
- Apply the same conditional-progression rules described in Section 5 to drive an equivalent terminal-based WHAT → WHEN → WHERE → task-details flow.

Because the model has no Astro-specific fields, no adapter layer is required beyond parsing JSON and applying Section 5's id-matching rules.

## 9. Examples

See `docs/public/schemas/wizard-data-model.example.json` for a complete example instance covering three goal categories (issue automation, scheduled report, PR review assistant), their compatible triggers/destinations, and a sample `promptTemplate`.

## References

- [Safe Outputs MCP Gateway Specification](/gh-aw/specs/safe-outputs-specification/)
- Parent issue: [#53498 — AW wizard in docs](https://github.com/github/gh-aw/issues/53498)
- `docs/public/schemas/mcp-gateway-config.schema.json` (existing schema convention this specification follows)

## Change Log

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-08-17 | Initial draft: goal categories, trigger options, destination options, prompt template, conditional progression, and future CLI reuse guidance. |
