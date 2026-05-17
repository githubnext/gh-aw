---
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

This skill converts markdown-based skill artifacts into a structured Scheduling-Structural-Logical (SSL) representation as described in the paper "Scheduling-Structural-Logical Representation for Agent Skills".

The skill extracts:

- scheduling metadata
- scene-level execution structure
- logic-step execution graphs

The output must be schema-valid, source-grounded, and conservatively normalized.

---

# Behavioral Requirements

## General Rules

- Only extract information directly supported by the source artifact.
- Do not invent hidden behavior, tools, dependencies, or side effects.
- Use restricted enum vocabularies only.
- Reject malformed outputs instead of silently repairing them.
- Prefer null, empty arrays, or coarse-grained classifications when evidence is weak.

---

# SSL Output Structure

## Top-Level Fields

The generated JSON must contain:

- `scheduling`
- `scenes`
- `logic_steps`

---

# Restricted Enumerations

## Allowed Scene Types

- `PREPARE`
- `ACQUIRE`
- `REASON`
- `ACT`
- `VERIFY`
- `RECOVER`
- `FINALIZE`

## Allowed Action Types

- `READ`
- `SELECT`
- `COMPARE`
- `VALIDATE`
- `INFER`
- `WRITE`
- `UPDATE_STATE`
- `CALL_TOOL`
- `REQUEST`
- `TRANSFER`
- `NOTIFY`
- `TERMINATE`

## Allowed Resource Scopes

- `MEMORY`
- `LOCAL_FS`
- `CODEBASE`
- `PROCESS`
- `USER_DATA`
- `CREDENTIALS`
- `NETWORK`
- `OTHER`

## Allowed Terminal Targets

### Scene Targets

- `END_SUCCESS`
- `END_FAIL`

### Logic-Step Targets

- `YIELD_SUCCESS`
- `YIELD_FAIL`

---

# Execution Pipeline

## Pass 1: Scheduling Extraction

Extract:

- skill identifier
- skill name
- skill goal
- intent signature
- expected inputs
- expected outputs
- dependencies
- control-flow features
- entry scene
- subscene references

### Requirements

- Use only explicit evidence from the source document.
- Preserve semantic intent without paraphrasing behavior into unsupported claims.
- Normalize identifiers consistently.

---

## Pass 2: Scene Decomposition

Break the skill into macro-level scenes.

### Requirements

- Prefer 2–5 scenes when supported by the source.
- Assign only allowed scene types.
- Define:
  - scene goals
  - entry conditions
  - exit conditions
  - next-scene rules
  - scene inputs
  - scene outputs

### Constraints

- Scene transitions must resolve to:
  - another scene ID
  - `END_SUCCESS`
  - `END_FAIL`

---

## Pass 3: Logic-Step Expansion

Expand scenes into atomic operational steps.

### Requirements

- Split logic steps when:
  - action type changes
  - resource boundary changes
  - execution effect changes
  - control-flow behavior changes
- Assign only allowed action types.
- Assign only allowed resource scopes.
- Use `$`-prefixed variable bindings.

### Data-Flow Rules

Use examples such as:

- `$user_request`
- `$selected_file`
- `$generated_output`

Do not use free-form unnamed intermediate variables.

---

## Pass 4: Validation

Validate:

- JSON syntax
- required fields
- enum membership
- unique identifiers
- graph integrity
- transition validity
- entry pointer validity
- scene containment
- logic-step containment

### Failure Handling

- Retry malformed generations within a bounded retry budget.
- Record validation failures explicitly.
- Reject records that remain invalid after retries.

---

# Validation Expectations

## Scheduling Validation

Ensure:

- entry scene exists
- referenced subscenes exist
- dependencies are normalized

## Scene Validation

Ensure:

- scene IDs are unique
- scene types are valid
- transition targets resolve correctly
- entry logic step exists

## Logic-Step Validation

Ensure:

- logic step IDs are unique
- action types are valid
- resource scopes are valid
- transition targets resolve correctly

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

Include per-artifact diagnostics.

Do not expose secrets or credentials in reports.

---

# Success Criteria

The skill succeeds when:

- a valid SSL JSON artifact is produced
- all references resolve correctly
- all enum values are valid
- the output passes schema validation
- the output remains grounded in the source artifact

The skill fails when:

- required graph structures are missing
- transitions are invalid
- unsupported inference is required
- validation errors remain unresolved after retries

---

# Output Expectations

## Primary Output

A schema-valid SSL JSON file named `ssl.json` placed alongside the source `SKILL.md`.

## Secondary Output

A validation and normalization report summarizing:

- accepted artifacts
- rejected artifacts
- validation diagnostics
- retry behavior

---

# Safety Constraints

- Never invent credentials or external systems.
- Never infer unstated side effects.
- Never fabricate execution logic.
- Never silently repair invalid graph structures.
- Never emit malformed JSON intentionally.
- Keep normalization deterministic where possible.

---

# Reuse Instructions

To apply this skill to a SKILL.md artifact:

1. Invoke this skill with `skill_path` pointing to the target `SKILL.md`.
2. The normalizer runs all four passes in sequence.
3. The resulting `ssl.json` is written alongside the source file.
4. Review the `validation_report` output to confirm acceptance.

For batch normalization, invoke this skill once per artifact and aggregate the per-artifact reports.
