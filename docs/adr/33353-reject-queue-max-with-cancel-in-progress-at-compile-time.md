# ADR-33353: Reject `queue: max` with `cancel-in-progress: true` at Compile Time

**Date**: 2026-05-19
**Status**: Draft
**Deciders**: @pelikhan, @copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

GitHub Actions does not permit `queue: max` in combination with `cancel-in-progress: true` on a concurrency block — the two settings are semantically incompatible (a run cannot both be cancelled and queued to the maximum). Without compile-time validation in `gh-aw`, users only discover this conflict when GitHub Actions rejects the workflow at runtime, after the workflow has already been compiled, committed, and pushed. The compiler already performs other concurrency validations (group expression syntax, parentheses balancing) in `pkg/workflow/concurrency_validation.go`, so this incompatibility is a natural addition. The check applies to both the top-level `concurrency` block and the `engine.concurrency` block, since either can independently introduce the invalid pair.

### Decision

We will reject the combination of `queue: max` and `cancel-in-progress: true` at compile time, in both workflow-level and engine-level concurrency blocks. The check is implemented as `validateConcurrencyQueueConfiguration` in `pkg/workflow/concurrency_validation.go` and is invoked from `validateToolConfiguration` in `pkg/workflow/compiler_validators.go` before the existing concurrency group expression validation runs. The error message names the forbidden pair and includes remediation hints (set `queue` to `single` or omit it, or set `cancel-in-progress: false`).

### Alternatives Considered

#### Alternative 1: Defer to GitHub Actions runtime validation

We could continue letting GitHub Actions reject the invalid combination at run time. This is the prior behavior. It was rejected because the feedback loop is much slower (compile → commit → push → run → fail), the resulting error is opaque to users who don't know which two fields conflict, and `gh-aw`'s value proposition is to catch this class of configuration error before the workflow ever reaches GitHub.

#### Alternative 2: Parse the concurrency YAML structurally instead of regex matching

We could unmarshal the serialized concurrency YAML into a typed struct (`Concurrency { Group, CancelInProgress, Queue }`) and check the fields by name. This was rejected for now because the rest of `concurrency_validation.go` already uses regex patterns over the serialized YAML (e.g. `concurrencyGroupPattern`), keeping consistency reduces blast radius, and the regex precisely targets the two keys we care about. Structural parsing remains an option if the set of cross-field checks grows.

#### Alternative 3: Silently rewrite the invalid combination

We could drop `queue: max` (or flip `cancel-in-progress`) when the conflict is detected. This was rejected because silently mutating a user-authored concurrency policy would mask intent and could change scheduling semantics in ways the author did not authorize.

### Consequences

#### Positive
- Invalid concurrency configurations are caught at compile time with a clear, actionable error rather than at GitHub Actions runtime.
- The check is uniformly applied to both top-level `concurrency` and `engine.concurrency`, eliminating one source of late-binding configuration errors.
- The error message names the conflicting pair and gives two concrete remediation paths, reducing time-to-fix.

#### Negative
- Regex-based detection over the serialized YAML can in principle miss exotic spellings (e.g. queue value spread across multiple lines, unusual quoting). The patterns currently cover unquoted, single-quoted, and double-quoted forms but are not a full YAML parser.
- Adds another compile-time check that must be kept in sync if GitHub Actions later changes which queue/cancel combinations are valid.

#### Neutral
- The validation runs before the existing group-expression validation, so this conflict surfaces first when both errors are present in the same block.
- Existing tests gain a new pattern (`TestValidateConcurrencyQueueConfiguration` unit tests plus integration cases for workflow-level and engine-level blocks) that future concurrency cross-field checks can follow.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Concurrency Configuration Validation

1. The compiler **MUST** reject any workflow whose top-level `concurrency` block contains both `queue: max` and `cancel-in-progress: true`.
2. The compiler **MUST** reject any workflow whose `engine.concurrency` block contains both `queue: max` and `cancel-in-progress: true`.
3. The validation **MUST** treat the quoted forms `queue: "max"` and `queue: 'max'` as equivalent to the unquoted `queue: max` for the purposes of this check.
4. The validation **MUST NOT** mutate the user-authored concurrency configuration; it **MUST** only report an error.
5. The validation **MUST** run before the concurrency group expression validation, so the conflict surfaces first when both errors are present.
6. The error returned **MUST** name both fields involved in the conflict (`queue: max` and `cancel-in-progress: true`) and **SHOULD** include a remediation hint pointing to at least one of: setting `queue` to a non-`max` value, omitting `queue`, or setting `cancel-in-progress: false`.
7. The validation **MUST** be a no-op when the serialized concurrency YAML is empty or contains only whitespace.
8. The validation **SHOULD** treat the concurrency YAML as opaque text matched by anchored patterns rather than fully parsing it, to remain consistent with sibling validators in `concurrency_validation.go`.
9. Implementations **MAY** later replace the regex-based detection with structural YAML parsing if the set of cross-field concurrency checks grows beyond what regex matching can cleanly express.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26110244263) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
