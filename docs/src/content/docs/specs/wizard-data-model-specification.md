---
title: Wizard Data Model Specification
description: Formal specification for the JSON data model that drives the docs wizard (WHAT → WHEN → WHERE → task details) and is designed for reuse by a future gh aw CLI consumer.
sidebar:
  order: 1371
---

# Wizard Data Model Specification

**Version**: 1.1.0
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
- The normative contract for assembling a final runnable prompt, including validation, step transitions, empty-state behavior, and completion criteria.

This specification does not define the Astro component implementation, CSS/visual presentation, or any styling details of the generated prompt preview — those are implementation details left to the docs site.

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

### 6.1 Normative Assembly Contract

The final prompt is the primary output of the wizard. A conforming implementation MUST assemble exactly one final prompt string from four user inputs plus bootstrap guidance:

1. **WHAT**: the selected `goalCategory`.
2. **WHEN**: the selected `triggerOption` allowed by that goal.
3. **WHERE**: the selected or inferred `destinationOption` allowed by that goal.
4. **Task details**: the user's free-text description.
5. **Bootstrap instructions**: fixed text that explains how to install and run `gh aw` if it is not already available.

The assembled prompt MUST be ordered as follows, regardless of visual UI layout:

1. Self-contained bootstrap/setup instructions.
2. Goal summary derived from WHAT.
3. Trigger summary derived from WHEN.
4. Destination summary derived from WHERE.
5. Free-text task details.
6. Final execution request instructing the downstream model to produce a runnable gh-aw workflow.

The prompt MUST be self-contained. In particular, its wording MUST NOT assume the user already has `gh`, `gh aw`, or any local gh-aw template files installed. The prompt MUST instruct the downstream model to include any necessary install/bootstrap guidance inside its answer.

When `promptTemplate.sections` is present, implementations SHOULD use its section order and headings if that order preserves the required semantic sequence above. Implementations MUST ignore any `promptTemplate.sections` ordering that would place task details before the required bootstrap/setup context or omit any of the four required answer groups.

### 6.2 Required Prompt Template

Conforming implementations MUST generate a prompt equivalent to the following template, with bracketed tokens replaced by resolved values:

```text
Create a GitHub Agentic Workflow (gh-aw) for this repository.

Assume nothing is preinstalled. Your answer must be self-contained and MUST include:
- how to install GitHub CLI if needed,
- how to install or run the gh-aw extension if needed,
- the complete workflow markdown file content,
- any required frontmatter and safe-outputs configuration,
- and brief usage instructions for a first-time user.

WHAT
- Goal: [goal label]
- Context: [goal help text or concise derived summary]

WHEN
- Trigger: [trigger label]
- Trigger intent: [plain-language summary of trigger frontmatter intent]

WHERE
- Destination: [destination label]
- Output intent: [plain-language summary of safe-output destination intent]

TASK DETAILS
[user free-text description]

Generate one runnable gh-aw workflow that satisfies the WHAT/WHEN/WHERE requirements above. If a required detail is still missing, say exactly what is missing instead of guessing.
```

The following is a concrete example of a fully assembled prompt string:

```text
Create a GitHub Agentic Workflow (gh-aw) for this repository.

Assume nothing is preinstalled. Your answer must be self-contained and MUST include:
- how to install GitHub CLI if needed,
- how to install or run the gh-aw extension if needed,
- the complete workflow markdown file content,
- any required frontmatter and safe-outputs configuration,
- and brief usage instructions for a first-time user.

WHAT
- Goal: Scheduled report
- Context: Generate a recurring workflow that gathers information on a schedule and publishes a result for humans to review.

WHEN
- Trigger: On a schedule
- Trigger intent: Run automatically from a cron-based schedule defined in workflow frontmatter.

WHERE
- Destination: Create or update a GitHub issue
- Output intent: Deliver the result by creating or updating an issue using safe outputs.

TASK DETAILS
Every Monday at 09:00 UTC, collect open pull requests labeled needs-review, summarize how long they have been waiting, and post a weekly triage report with sections for urgent items, stale items, and links to each PR.

Generate one runnable gh-aw workflow that satisfies the WHAT/WHEN/WHERE requirements above. If a required detail is still missing, say exactly what is missing instead of guessing.
```

An implementation MAY vary surrounding prose, capitalization, or bullet style, but it MUST preserve the same semantic content and MUST include the self-contained bootstrap requirements.

### 6.3 Validation Rules for Incomplete Input

The wizard MUST validate user input before producing the final prompt. It MUST NOT assemble a final prompt if any required input is missing or invalid.

The following validation rules are REQUIRED:

| Field | Requirement | Failure behavior |
|---|---|---|
| WHAT / `goalCategory` | REQUIRED. Exactly one allowed goal MUST be selected. | Disable forward progression past WHAT and show an inline validation message. |
| WHEN / `triggerOption` | REQUIRED. Exactly one trigger compatible with the selected goal MUST be selected. | Disable forward progression past WHEN and show an inline validation message. |
| WHERE / `destinationOption` | REQUIRED before completion. It MAY be auto-inferred, but the resulting destination value MUST be explicit in wizard state. | Disable completion and prompt rendering until the destination is inferred or selected. |
| Task details | REQUIRED non-empty trimmed string. | Disable completion and prompt rendering; show an inline validation message. |

Additional validation requirements:

- A destination inferred from trigger type is provisional until it resolves to a destination ID that is allowed by the selected goal's `destinationOptionIds`.
- If inference produces zero matches or multiple equally valid matches, the wizard MUST require explicit user selection at the WHERE step.
- A task-details value containing only whitespace MUST be treated as empty.
- Implementations SHOULD require enough text to avoid a vacuous prompt. A minimum of 10 non-whitespace characters is RECOMMENDED.
- If WHAT changes and the existing WHEN or WHERE selection is no longer compatible, those downstream values MUST be cleared before validation runs.
- If WHEN changes and the existing WHERE selection is no longer compatible or no longer inferable, the WHERE value MUST be cleared and re-requested.

The wizard MUST fail closed: it is better to block prompt generation than to emit a prompt with a missing destination, missing task details, or stale incompatible selections.

### 6.4 Step Transitions and Back/Next Behavior

The canonical flow is WHAT → WHEN → WHERE → task details.

Implementations MUST follow these transition rules:

1. **Initial entry**: only the WHAT step is active; later steps are disabled or visually unavailable until prerequisite data exists.
2. **Advancing from WHAT to WHEN**: allowed only after a valid WHAT selection exists.
3. **Advancing from WHEN to WHERE**: allowed only after a valid WHEN selection exists.
4. **Advancing from WHERE to task details**: allowed only after a valid WHERE value exists, whether explicitly selected or inferred.
5. **Completion from task details**: allowed only after all prior steps remain valid and task details pass validation.

Back/next semantics are normative:

- **Next** MUST run validation for the current step and MUST NOT advance if the step is incomplete.
- **Back** MUST preserve earlier valid answers and MUST allow the user to revise them.
- Revisiting an earlier step and changing its answer MUST trigger downstream revalidation immediately.
- If a downstream answer remains valid after an upstream change, it MAY be preserved.
- If a downstream answer becomes invalid after an upstream change, it MUST be cleared and the user MUST be returned to the earliest now-invalid step before completion is allowed.

Required reset examples:

- Changing WHAT from a goal that supports `schedule` to one that does not MUST clear any previously selected scheduled WHEN option.
- Changing WHAT to a goal with a different destination set MUST clear an incompatible WHERE selection.
- Changing WHEN from `issues` to `schedule` MAY replace an inferred comment destination with an inferred issue destination, but only if the new destination is uniquely valid for the current goal.

### 6.5 Empty-State and Blocking-State Handling

Before any selection is made, the wizard is in an empty state.

In the empty state, implementations MUST:

- Render the WHAT step with no selected answer.
- Present later steps as unavailable, disabled, or clearly marked "Complete the previous step first."
- Avoid showing a final prompt preview, because no conforming prompt can yet be assembled.

If a user attempts to advance without a required answer, the UI MUST:

- keep focus on the current step,
- display a clear inline error message near the missing control,
- avoid generating or updating a misleading "partial final prompt", and
- preserve any earlier valid answers already given.

Recommended empty/blocking copy includes:

- WHAT not selected: "Choose what this workflow should do before continuing."
- WHEN not selected: "Choose when the workflow should run before continuing."
- WHERE unresolved: "Choose where the workflow should send its result before continuing."
- Task details empty: "Describe the task in enough detail to generate a workflow."

Implementations MAY show a draft preview before completion, but any such preview MUST be clearly labeled incomplete and MUST NOT be represented as runnable until all completion criteria are met.

### 6.6 Completion Criteria

The wizard is "done" only when all of the following are true:

1. A valid WHAT selection exists.
2. A valid WHEN selection exists and is compatible with WHAT.
3. A valid WHERE value exists, whether selected or inferred, and is compatible with WHAT and WHEN.
4. Task details exist as a non-empty trimmed string.
5. The final prompt string has been assembled with the required self-contained bootstrap/setup wording.

Only after all five conditions are satisfied MAY the implementation:

- render the final prompt as complete,
- enable any copy/export action for that prompt, and
- describe the prompt as runnable.

If any of the above conditions later becomes false because the user edits an earlier step, the wizard MUST revert to an incomplete state and MUST re-run validation before allowing completion again.

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
| 1.1.0 | 2026-08-18 | Added concrete prompt assembly contract: template composition example, input validation rules, step transition/back-next behavior, empty-state handling, and completion criteria (#53515). |
| 1.0.0 | 2026-08-17 | Initial draft: goal categories, trigger options, destination options, prompt template, conditional progression, and future CLI reuse guidance. |
