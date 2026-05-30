# ADR-29995: Compile-Time Validation of the Model Alias Format Specification

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The repository defines a Model Alias Format (MAF) specification (`docs/src/content/docs/reference/model-alias-specification.md`) that governs the syntax of model identifier strings used in workflow frontmatter (`models:` map) and the `engine.model` field. Prior to this change, the spec existed as documentation only: invalid identifiers (bad characters, circular aliases, out-of-range parameters, globs where forbidden) would either be silently accepted at compile time and fail at runtime, or propagate as confusing downstream errors. The workflow compiler already had a validation extension point (`ParseWorkflowFile`), making it a natural site for enforcement. The spec itself defines discrete, numbered validation rules (V-MAF-001 through V-MAF-013), making a rule-by-rule implementation tractable.

### Decision

We will enforce the Model Alias Format specification at compile time by wiring an ABNF grammar parser (`ParseModelIdentifier`) and a set of validation functions (`validateModelAliasMap`) into `ParseWorkflowFile`, immediately after the model mappings are populated. Violations of hard rules (V-MAF-001 through V-MAF-010) abort compilation with a specific error message naming the offending character and rule code. Soft rules (V-MAF-011) emit warnings. GitHub Actions expression strings (`${{ … }}`) are exempt from syntax checks because they are resolved at runtime. This approach front-loads error detection at the earliest possible phase, providing actionable feedback before any agent work begins.

### Alternatives Considered

#### Alternative 1: Runtime Validation at Agent Invocation

Validate model identifiers only when an agent is actually dispatched and the model is resolved. This would catch all the same errors but only after the workflow has started executing, meaning partial work may have already been done (steps executed, tokens consumed) before the bad identifier surfaces. It was rejected because the compiler already parses the full workflow graph before execution; shifting validation left is strictly better for the operator experience and aligns with the existing compiler's role as the semantic gatekeeper.

#### Alternative 2: JSON Schema / YAML Schema Validation

Encode the MAF grammar constraints as a JSON Schema attached to the workflow file format and validate at schema-validation time. This approach is declarative and tooling-friendly (editors can surface errors inline) but cannot express recursive constraints such as circular alias detection (V-MAF-010) or contextual rules such as "globs are allowed in alias entries but not in `engine.model`" (V-MAF-004). It was rejected because the MAF spec's constraint set exceeds what JSON Schema can express without custom keywords.

#### Alternative 3: No Enforcement — Spec as Documentation Only

Leave the spec as normative documentation and rely on code review and convention to prevent violations. This was rejected because experience showed that undocumented or leniently enforced identifier formats accumulate technical debt: callers begin passing provider-scoped globs to `engine.model`, or create circular aliases that are only caught in production, or use undocumented parameter keys that silently have no effect.

### Consequences

#### Positive
- Malformed model identifiers are caught at the earliest possible phase, before any agent work begins, producing clear error messages that name the offending character and the violated rule code (e.g., V-MAF-004).
- Circular alias chains — a subtle class of infinite-loop bugs — are now detected deterministically at compile time via DFS cycle detection.
- The spec's numbered validation rules map 1:1 to code, making it straightforward to verify coverage and add new rules as the spec evolves.

#### Negative
- Workflows that previously relied on leniently accepted (but technically invalid) model identifiers will now fail to compile; operators must update their frontmatter to conform to the spec.
- Adding a validation pass to `ParseWorkflowFile` increases compile-time latency proportional to the number of alias entries; for very large alias maps this may be measurable.

#### Neutral
- GitHub Actions expressions in model identifier fields (`${{ inputs.model }}`) must be explicitly detected and skipped, adding a dependency on an `isExpression` helper that must be kept in sync with the GHA expression syntax.
- Validation rules V-MAF-012 and V-MAF-013 (effort warning for non-reasoning models; runtime cycle guard) are explicitly out of scope for compile-time enforcement and must remain as runtime/agent-side checks.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Model Identifier Parsing

1. Implementations **MUST** parse every model identifier string using the ABNF grammar defined in Section 4.1 of the Model Alias Format Specification before the identifier is used in any compilation or execution context.
2. Implementations **MUST NOT** accept a model identifier that contains a character outside the allowed set for its segment type; the resulting error message **MUST** name the offending character and the segment type (V-MAF-001, V-MAF-006).
3. Implementations **MUST NOT** accept an empty model identifier string.
4. Implementations **SHOULD** exempt strings matching the GitHub Actions expression pattern (`${{ … }}`) from syntax validation, as their content is resolved at runtime.

### Parameter Value Validation

1. Implementations **MUST** reject any model identifier whose `effort` parameter value is not one of `low`, `medium`, or `high` (V-MAF-002).
2. Implementations **MUST** reject any model identifier whose `temperature` parameter value cannot be parsed as a decimal float in the range `[0.0, 2.0]` inclusive (V-MAF-003).
3. Implementations **SHOULD** emit a warning for each unrecognised parameter key encountered in a model identifier; implementations **MUST NOT** treat an unrecognised parameter key as a hard error (V-MAF-011).

### Alias Key Constraints

1. Implementations **MUST NOT** accept an alias map key that contains the characters `/`, `?`, or `&` (V-MAF-005).
2. Implementations **MAY** accept the empty string as an alias key; it is interpreted as the default policy alias.

### Engine Model Constraints

1. Implementations **MUST NOT** accept a glob pattern (an identifier containing `*`) as the value of `engine.model` (V-MAF-004).
2. Implementations **MUST** validate the syntax of a non-empty `engine.model` value using the same ABNF grammar applied to alias entries.

### Circular Alias Detection

1. Implementations **MUST** perform a depth-first cycle check over the fully merged alias map after all alias layers (builtins, imports, frontmatter) have been combined (V-MAF-010).
2. Implementations **MUST** abort compilation if a cycle is detected; the error message **MUST** name every alias key involved in the cycle.
3. Implementations **MUST NOT** follow provider-scoped names (identifiers containing `/`) or glob patterns as alias references during cycle detection.

### Compile-Time Integration

1. Implementations **MUST** invoke model alias validation within `ParseWorkflowFile` immediately after `ModelMappings` is populated, before returning the compiled `WorkflowData`.
2. Implementations **MUST** propagate any hard-validation error as the return value of `ParseWorkflowFile`, preventing further compilation.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25288249489) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
