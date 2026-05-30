# ADR-29212: Allow GitHub Actions Expression Inputs for List-Valued Safe-Output Constraints

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Safe-output fields such as `labels`, `allowed-repos`, and `allowed-base-branches` define list-valued runtime constraints on safe-output actions (`push-to-pull-request-branch`, `create-pull-request`, `add-comment`). These fields previously required literal YAML arrays, which worked well for single-repository workflows but made reusable `workflow_call` workflows impossible to parameterize: callers cannot inject list values at runtime because the parser only accepted static YAML sequences. The demand for reusable workflows that delegate constraint configuration to their callers surfaces this limitation as a hard blocker.

### Decision

We will extend the six affected list-valued safe-output fields to accept either a literal YAML array (existing behavior, unchanged) or a single GitHub Actions expression string (e.g., `${{ inputs.required-labels }}`). At build time the compiler wraps a detected expression in a single-element `[]string` slice so the existing struct fields survive YAML unmarshal; it then emits the expression as a raw JSON string into the generated config so that GitHub Actions evaluates it inside the heredoc before `config.json` is written. Non-expression bare strings are rejected with an actionable error. No changes are required to the JavaScript handlers because they already accept comma-separated strings.

### Alternatives Considered

#### Alternative 1: Keep Requiring Literal Arrays (Status Quo)

Literal arrays remain the only accepted value form. This was rejected because it forces workflow authors to hardcode list constraints, making reusable `workflow_call` workflows impossible to parameterize — the direct trigger for this PR.

#### Alternative 2: Add New Expression-Specific Sibling Fields

Introduce new fields (e.g., `labels-expr`, `allowed-repos-expr`) that accept expression strings, leaving the existing array fields untouched. This was rejected because it doubles the number of schema fields, creates confusing dual-field semantics, and requires callers to learn a new naming convention for every parameterizable list field.

#### Alternative 3: Resolve Expressions at Compile Time

Attempt to evaluate `workflow_call` input defaults at compile time to produce a concrete array. This was rejected because `workflow_call` inputs are only known at runtime when the calling workflow passes arguments; compile-time resolution is not feasible for this class of values.

### Consequences

#### Positive
- Reusable `workflow_call` workflows can now parameterize any of the six list-valued constraint fields, unblocking a common workflow composition pattern.
- Literal YAML array behavior is fully preserved; no migration is required for existing workflows.
- JavaScript runtime handlers require no changes because they already parse comma-separated strings.

#### Negative
- JSON schema for the six fields changes from `type: "array"` to `oneOf: [{type: "array"}, {type: "string"}]`, reducing compile-time strictness: a mis-typed bare string now produces a runtime error rather than a parse error.
- Expression correctness (e.g., whether the referenced input exists) is validated only at runtime, not at compile time.
- The new `preprocessStringArrayFieldAsTemplatable` helper adds a pre-pass over `configData` that must be kept in sync whenever new list-valued fields are added.

#### Neutral
- The PR number is used as the ADR number (`29212`), consistent with the project's ADR numbering convention.
- Six fields are affected across three parsers: `labels` and `allowed-repos` in `push_to_pull_request_branch.go`, `labels`, `allowed-repos`, and `allowed-base-branches` in `create_pull_request.go`, and `allowed-repos` in `add_comment.go`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Expression Detection and Preprocessing

1. Implementations **MUST** accept either a YAML sequence (literal array) or a single GitHub Actions expression string of the form `${{ ... }}` as the value of any list-valued safe-output constraint field (`labels`, `allowed-repos`, `allowed-base-branches`).
2. Implementations **MUST NOT** accept a bare string value that is not a valid GitHub Actions expression (i.e., does not match `^\$\{\{.*\}\}$`); such values **MUST** produce a descriptive parse error.
3. Implementations **MUST** detect expression strings in `configData` before YAML unmarshal and wrap them in a single-element `[]string` slice so that existing `[]string` struct fields survive unmarshalling unchanged.
4. Implementations **MUST NOT** attempt to evaluate or validate the content of an expression at compile time.

### Compiler Emission

1. Implementations **MUST** emit a detected single-element expression slice as a raw JSON string (not a JSON array) in the generated config heredoc, so that GitHub Actions evaluates the expression before `config.json` is written.
2. Implementations **MUST** emit a literal multi-element slice or a slice containing no expression as a JSON array.
3. Implementations **SHOULD** use `AddTemplatableStringSlice` (or an equivalent abstraction) for all six affected fields when constructing the handler config, rather than calling `AddStringSlice` directly.

### JSON Schema

1. The JSON Schema entry for each of the six affected fields **MUST** be changed from `type: "array"` to `oneOf: [{type: "array"}, {type: "string"}]` to accept both literal arrays and expression strings.
2. The array variant within `oneOf` **MUST** retain the same `items` sub-schema that was present before this change.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25140707660) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
