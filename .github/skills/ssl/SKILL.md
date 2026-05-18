---
# This frontmatter describes a meta-skill for the SSL Skill Normalizer.
# It defines the skill's interface (inputs/outputs) and required tools.
# It is not an executable gh-aw workflow; it is a reusable skill artifact
# invoked by agents that implement the SSL normalization pipeline.
name: ssl-skill-normalizer
description: Normalize SKILL.md artifacts into Scheduling-Structural-Logical (SSL) JSON representations using a conservative multi-pass extraction pipeline.
tools:
  - read_file
  - write_file
  - search_files
  - json_validate
  - create_artifact
  - run_tests
inputs:
  - skill_path
outputs:
  - ssl_json
  - validation_report
---

# SSL Skill Normalizer

## Purpose

This skill converts markdown-based skill artifacts into a structured **Scheduling-Structural-Logical (SSL)** representation as introduced in:

> Liang et al., "From Skill Text to Skill Structure: The Scheduling-Structural-Logical Representation for Agent Skills", arXiv:2604.24026 (2026).

SSL addresses the core limitation of free-form skill text: it is human-readable but hard for agents to reason over, discover, and audit. By mapping each skill into three complementary layers, SSL makes skills **searchable** (improved MRR 0.573 → 0.707 in the paper) and **risk-assessable** (improved macro F1 0.744 → 0.787).

---

# The Three SSL Layers

The representation is grounded in Schank & Abelson's theories of Memory Organization Packets (MOPs), Script Theory, and Conceptual Dependency. Each layer captures a different dimension of skill knowledge:

## Layer 1 — Scheduling (When / Who)

Answers: *When should this skill be invoked? By whom, given which inputs and outputs?*

Fields extracted:
- `id` — stable lowercase identifier
- `name` — human-readable skill name
- `goal` — one-sentence purpose
- `intent_signature` — typed function signature (`fn($input) -> $output`)
- `inputs` — `$`-prefixed named input bindings
- `outputs` — `$`-prefixed named output bindings
- `dependencies` — explicit runtime tool or library requirements
- `control_flow_features` — e.g. `sequential`, `conditional`, `loop`
- `entry_scene` — ID of the first scene to execute
- `subscene_refs` — IDs of any nested/delegated scenes

## Layer 2 — Structural (How / Order)

Answers: *What are the macro-level execution stages and how do they connect?*

Each **scene** is a named execution stage with:
- `id` — unique within the skill
- `type` — one of the restricted scene-type enum (see below)
- `goal` — what the scene accomplishes
- `entry_condition` — precondition for entering the scene
- `exit_condition` — postcondition that must hold on exit
- `next_scene_rules` — conditional transitions to the next scene ID, `END_SUCCESS`, or `END_FAIL`
- `inputs` / `outputs` — `$`-prefixed bindings consumed and produced
- `entry_logic_step` — ID of the first logic step in this scene

## Layer 3 — Logical (What / Actions)

Answers: *What atomic operations are performed, on which resources?*

Each **logic step** is an indivisible operation with:
- `id` — unique within the skill
- `scene_id` — owning scene
- `action_type` — one of the restricted action-type enum (see below)
- `resource_scope` — one of the restricted resource-scope enum (see below)
- `description` — one sentence describing the operation
- `inputs` / `outputs` — named `$`-variable bindings
- `next` — ID of the following step, `YIELD_SUCCESS`, or `YIELD_FAIL`

---

# Restricted Enumerations

## Scene Types

| Value | Meaning |
|---|---|
| `PREPARE` | Setup: load inputs, configure environment |
| `ACQUIRE` | Receive or fetch required data |
| `REASON` | Analyze, infer, or plan |
| `ACT` | Produce or transform primary output |
| `VERIFY` | Validate outputs or preconditions |
| `RECOVER` | Handle failure; retry or compensate |
| `FINALIZE` | Write results, emit notifications, clean up |

## Action Types

| Value | Meaning |
|---|---|
| `READ` | Consume data from a resource without side effects |
| `SELECT` | Choose among alternatives |
| `COMPARE` | Diff or rank two or more values |
| `VALIDATE` | Assert a constraint or schema |
| `INFER` | Derive new information via reasoning |
| `WRITE` | Produce or overwrite data in a resource |
| `UPDATE_STATE` | Mutate shared state |
| `CALL_TOOL` | Invoke an external tool or subprocess |
| `REQUEST` | Send a request to an external service |
| `TRANSFER` | Move data between resources |
| `NOTIFY` | Emit a message or event |
| `TERMINATE` | End execution and return control |

## Resource Scopes

| Value | Meaning |
|---|---|
| `MEMORY` | In-process working memory |
| `LOCAL_FS` | Local file system |
| `CODEBASE` | Source code under version control |
| `PROCESS` | OS process or shell |
| `USER_DATA` | User-provided or personal data |
| `CREDENTIALS` | Secrets, tokens, or credentials |
| `NETWORK` | Remote network resource |
| `OTHER` | Any resource not covered above |

## Terminal Targets

- **Scene transitions**: `END_SUCCESS` | `END_FAIL`
- **Logic-step transitions**: `YIELD_SUCCESS` | `YIELD_FAIL`

---

# Behavioral Requirements

## General Rules

- Only extract information directly supported by the source artifact.
- Do not invent hidden behavior, tools, dependencies, or side effects.
- Use restricted enum vocabularies only; never free-form strings in typed fields.
- Reject malformed outputs instead of silently repairing them.
- Prefer `null`, empty arrays, or coarse-grained classifications when evidence is weak.

---

# Execution Pipeline

## Pass 1: Scheduling Extraction

Read the source `SKILL.md`, then extract the scheduling layer.

Produce `scheduling` with all fields in Layer 1. When evidence is absent for an optional field, emit an empty array or `null`.

**Requirements**
- Use only explicit evidence from the source document.
- Preserve semantic intent without paraphrasing behavior into unsupported claims.
- Normalize all identifiers to `snake_case`.

---

## Pass 2: Scene Decomposition

Analyse the skill's execution flow and decompose it into macro-level scenes.

**Requirements**
- Prefer 2–5 scenes when supported by the source. Only add more if the source describes clearly distinct phases.
- Assign only allowed scene types from the enum table.
- For each scene define: goal, entry_condition, exit_condition, next_scene_rules, inputs, outputs, entry_logic_step.

**Constraints**
- Every `next_scene_rules` target must resolve to another scene ID, `END_SUCCESS`, or `END_FAIL`.
- Include a `RECOVER` scene when the source describes retry or error-recovery behaviour.

---

## Pass 3: Logic-Step Expansion

Expand each scene into its sequence of atomic logic steps.

**Split a step whenever any of the following changes:**
- action type
- resource boundary
- execution effect
- control-flow behaviour

**Requirements**
- Assign only allowed action types and resource scopes.
- Use `$`-prefixed variable bindings for all named data (`$user_request`, `$selected_file`, `$generated_output`).
- Do not use unnamed or free-form intermediate variables.

---

## Pass 4: Validation

Validate the draft SSL JSON against all of the following rules:

| Rule | Check |
|---|---|
| JSON syntax | Well-formed JSON |
| Required fields | All top-level fields present |
| Enum membership | All enum fields use allowed values only |
| Unique identifiers | All scene IDs and step IDs are globally unique |
| Entry pointer | `entry_scene` references an existing scene ID |
| Scene entry pointer | `entry_logic_step` references an existing step ID |
| Scene containment | All referenced scene IDs exist |
| Logic-step containment | All referenced step IDs exist |
| Transition validity | All transition targets are valid scene/step IDs or terminal values |
| Graph integrity | No unreachable scenes or dangling references |

**Failure Handling**
- Retry malformed generations within a bounded retry budget (recommend ≤ 3 retries).
- Record each validation failure with the specific rule that was violated.
- Reject records that remain invalid after retries; do not silently emit invalid JSON.

---

# Reporting

Generate a normalization report containing:

- processed artifact count
- valid SSL count
- rejected SSL count
- parse failures
- schema failures
- graph failures
- enum failures
- retry counts

Include per-artifact diagnostics with the specific Pass-4 rule that caused rejection.

Do not expose secrets or credentials in reports.

---

# Success Criteria

The skill succeeds when:

- a valid SSL JSON artifact is produced
- all references resolve correctly
- all enum values are valid
- the output passes all Pass-4 validation rules
- the output remains grounded in the source artifact with no invented behaviour

The skill fails when:

- required graph structures are missing
- transitions are invalid
- unsupported inference is required to fill required fields
- validation errors remain unresolved after retries

---

# Output Expectations

## Primary Output

A schema-valid SSL JSON file named `ssl.json` placed alongside the source `SKILL.md`. Top-level keys: `scheduling`, `scenes`, `logic_steps`.

## Secondary Output

A validation and normalization report summarizing accepted artifacts, rejected artifacts, per-artifact validation diagnostics, and retry behaviour.

---

# Safety Constraints

- Never invent credentials or external systems.
- Never infer unstated side effects.
- Never fabricate execution logic not present in the source.
- Never silently repair invalid graph structures.
- Never emit malformed JSON intentionally.
- Keep normalization deterministic where possible.

---

# Reuse Instructions

To apply this skill to a SKILL.md artifact:

1. Invoke this skill with `skill_path` pointing to the target `SKILL.md`.
2. The normalizer runs all four passes in sequence.
3. If Pass 4 fails, the `RECOVER` pass retries generation up to the retry budget.
4. The resulting `ssl.json` is written alongside the source file.
5. Review the `validation_report` output to confirm acceptance.

For batch normalization, invoke this skill once per artifact and aggregate the per-artifact reports.
