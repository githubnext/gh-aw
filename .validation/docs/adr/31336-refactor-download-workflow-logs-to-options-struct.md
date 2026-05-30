# ADR-31336: Refactor `DownloadWorkflowLogs` to a Typed Options Struct

**Date**: 2026-05-10
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`DownloadWorkflowLogs` in `pkg/cli/logs_orchestrator.go` had grown to a 26-parameter positional signature, including several adjacent `bool` parameters (e.g., `verbose, toolGraph, noStaged, firewallOnly, noFirewall, parse, jsonOutput, ...`). Eight callsites — one production callsite in `pkg/cli/logs_command.go` and seven test callsites — passed arguments by position, making them brittle and hard to read. Adjacent same-typed parameters carry a real risk of silent mis-ordering that the compiler cannot catch, and any addition or reordering of parameters required mechanical churn at every callsite. The function's orchestration logic is otherwise intact and not the subject of this change.

### Decision

We will replace the positional parameter list of `DownloadWorkflowLogs` with a typed options struct, `LogsDownloadOptions`, and update the signature to `func DownloadWorkflowLogs(ctx context.Context, opts LogsDownloadOptions) error`. All existing callers will pass a named struct literal, allowing each call to specify only the fields it cares about while zero-valued fields preserve prior defaults. This is an interface refactor only — the orchestration logic, behavior, and field semantics are unchanged.

### Alternatives Considered

#### Alternative 1: Keep the positional signature

Leaving the 26-parameter signature in place avoids any code churn but preserves the legibility and correctness risks (silent mis-ordering between adjacent bools, churn at every callsite when a parameter is added). Given that callsites already required line-by-line `// fieldName` comments to remain readable, this was rejected as continuing to accumulate technical debt.

#### Alternative 2: Functional options pattern (`WithX(...)` constructors)

Using a chain of `Option` functions (`opts ...Option`) is idiomatic in some Go libraries and offers extensibility for defaults. It was rejected because the parameter set is a flat configuration bag with no need for hidden state, lazy evaluation, or extensibility hooks; a plain struct literal is more discoverable in IDEs, simpler to read, and aligns with how callers (CLI flag plumbing) already aggregate values.

#### Alternative 3: Split the function into smaller orchestrators

Decomposing `DownloadWorkflowLogs` into multiple smaller functions, each with fewer parameters, would address the symptom by reducing surface area. It was rejected for this PR because it is a logic rewrite rather than an interface refactor and would expand scope significantly; the options-struct change is a low-risk first step that does not preclude future decomposition.

### Consequences

#### Positive
- Callsites become self-documenting via field names; mis-ordering between same-typed parameters is no longer possible.
- Adding a new optional parameter is a non-breaking change for callers that don't use it (zero value preserves prior behavior).
- Test code becomes substantially shorter — only fields relevant to the test need to be set.

#### Negative
- The shim inside `DownloadWorkflowLogs` that re-binds every option field to a local variable preserves the existing function body verbatim but adds 25 trivial lines; future cleanup should reference `opts.X` directly and remove the shim.
- The public API of `pkg/cli` changes shape, so any out-of-tree caller (if any exist) must migrate.
- A 26-field struct is itself a smell; the struct documents the shape of the problem but does not solve it. Decomposition (Alternative 3) is still warranted as follow-up.

#### Neutral
- Field defaults are now expressed implicitly via Go zero values rather than explicit positional arguments; reviewers must verify that "omitted" matches "previously passed as zero."
- All known callers (1 production + 7 tests) have been migrated in this PR; no deprecation period is needed.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Function Signature

1. The exported function `DownloadWorkflowLogs` **MUST** have the signature `func DownloadWorkflowLogs(ctx context.Context, opts LogsDownloadOptions) error`.
2. The exported type `LogsDownloadOptions` **MUST** be a struct defined in `pkg/cli/logs_orchestrator.go`.
3. New configuration parameters added to the workflow log download flow **MUST** be added as fields on `LogsDownloadOptions` rather than as additional positional parameters.
4. `DownloadWorkflowLogs` **MUST NOT** reintroduce any positional configuration parameters beyond `ctx` and `opts`.

### Caller Conventions

1. Callers of `DownloadWorkflowLogs` **MUST** pass a `LogsDownloadOptions` value constructed as a named struct literal (i.e., `LogsDownloadOptions{Field: value, ...}`).
2. Callers **SHOULD** specify only the fields whose non-zero values are meaningful for the call; reliance on zero values to express defaults is **RECOMMENDED**.
3. Test code **SHOULD NOT** restate the full set of fields when a subset suffices.

### Field Semantics

1. Zero values for fields on `LogsDownloadOptions` **MUST** preserve the same behavior as the previous positional signature passed `0`, `""`, `false`, or `nil` for the corresponding argument.
2. Changes to the meaning of any existing field on `LogsDownloadOptions` **MUST** be recorded in a new ADR that supersedes or amends this one.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25633125088) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
