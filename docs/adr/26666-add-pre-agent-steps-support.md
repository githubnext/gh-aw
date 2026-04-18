# ADR-26666: Introduce `pre-agent-steps` as a Distinct Workflow Extension Point

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiler translates markdown workflow files into GitHub Actions YAML. It already supports two custom step extension points: `pre-steps` (injected before the checkout step, very early in the agent job) and `post-steps` (injected after AI engine execution). A gap exists for steps that must run after all framework initialization — including checkout, credential setup, and environment preparation — but *immediately before* the AI engine starts. Use cases include final context preparation scripts, last-moment environment variable injection, and validations that depend on checked-out code being available.

### Decision

We will add a new `pre-agent-steps` frontmatter field that is injected into the compiled GitHub Actions YAML immediately before the engine execution step, after all setup steps (including the CLI proxy start step). Like `pre-steps`, `pre-agent-steps` participates in the import merge system: imported `pre-agent-steps` are prepended before the main workflow's `pre-agent-steps`, preserving a deterministic, layered execution order. The existing `validateStepsSecrets` infrastructure is extended to cover the new field, applying identical secret expression restrictions.

### Alternatives Considered

#### Alternative 1: Overload the existing `steps` field with ordering semantics

The existing `steps` field (custom steps) could be repurposed or extended with explicit positioning metadata (e.g., `position: before-engine`). This was rejected because `steps` already has a defined execution context and adding positional metadata would complicate both the schema and the compiler without providing a clearer mental model. A dedicated named field communicates intent more directly.

#### Alternative 2: Extend `pre-steps` with a phase option

`pre-steps` could accept an optional `phase: post-setup` attribute to signal late-bound execution. This was rejected because it conflates two very different lifecycle positions (before checkout vs. just before the engine) into a single field, making the execution model harder to reason about. Keeping phase-specific fields separate aligns with how GitHub Actions itself structures job steps.

#### Alternative 3: Require users to use `post-steps` with negated conditions

Users could approximate "run before engine" behaviour by placing steps in `post-steps` with `if: always()` or condition expressions that short-circuit. This is semantically backwards, error-prone to write, and would not actually execute *before* the AI engine — it would run after, defeating the use case entirely.

### Consequences

#### Positive
- Users can inject pre-execution logic (context preparation, validation, environment mutations) at exactly the right lifecycle point — after setup, before the agent starts.
- The new field mirrors the naming and import-merge semantics of `pre-steps` and `post-steps`, keeping the mental model consistent and the feature discoverable.
- Secret expression validation is automatically enforced on the new field using the existing `validateStepsSecrets` infrastructure, preserving the security model without new code.
- Integration tests cover placement order (after clean-git-credentials, before engine execution) and import merge ordering.

#### Negative
- The workflow extension model now has three named step positions (`pre-steps`, `pre-agent-steps`, `post-steps`) plus the main `steps` field; the distinction between `pre-steps` and `pre-agent-steps` requires documentation to avoid confusion.
- Each new step type requires coordinated changes across the schema, type definitions, import extractor, compiler builder, YAML generator, serialization, and documentation — increasing the cost of future schema evolution.

#### Neutral
- The import merge ordering convention (imported steps prepend, main steps append) is the same as `pre-steps`; this is a neutral consistency consequence rather than a new design choice.
- The `docs/adr/` filename uses the PR number as the sequence identifier, consistent with existing ADR naming in this repository.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema

1. Workflows **MAY** define a `pre-agent-steps` top-level frontmatter field containing an array of GitHub Actions step definitions.
2. The `pre-agent-steps` field **MUST** conform to the same step schema as `post-steps` (object or string items, with `additionalProperties: true`).
3. Implementations **MUST NOT** treat `pre-agent-steps` and `pre-steps` as interchangeable; they represent different lifecycle positions and **MUST** be processed independently.

### Compilation and Placement

1. Implementations **MUST** inject all resolved `pre-agent-steps` into the compiled GitHub Actions YAML immediately before the engine execution step.
2. `pre-agent-steps` **MUST** be placed after all framework setup steps (including the CLI proxy start step) and **MUST NOT** be placed before any framework-owned initialization steps.
3. If no `pre-agent-steps` are defined (neither in the main workflow nor in any imports), the compiler **MUST NOT** emit any placeholder or empty steps block.

### Import Merge Ordering

1. When a workflow imports other workflows that also define `pre-agent-steps`, implementations **MUST** prepend imported `pre-agent-steps` before the main workflow's `pre-agent-steps` in the merged output.
2. Import merge order **MUST** follow the same topological ordering used for all other merged fields.

### Secret Expression Validation

1. Implementations **MUST** apply secret expression validation to the `pre-agent-steps` field using the same rules applied to `pre-steps`, `steps`, and `post-steps`.
2. In strict mode, secrets expressions in non-`env`/`with` contexts within `pre-agent-steps` **MUST** cause a compilation error.
3. In non-strict mode, secrets expressions in `pre-agent-steps` **SHOULD** emit a warning.

### Action Pinning

1. Implementations **MUST** apply action pin resolution (SHA substitution) to `uses:` references in `pre-agent-steps`, consistent with how pinning is applied to `pre-steps` and `post-steps`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: `pre-agent-steps` are injected immediately before the engine execution step (not before setup steps), imports are merged in prepend order, secret validation is applied, and action pinning is applied. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24519930482) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
