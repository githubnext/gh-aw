# ADR-29567: Decompose generateMainJobSteps and buildConsolidatedSafeOutputsJob into Phase Helpers

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/workflow/compiler_yaml_main_job.go` and `pkg/workflow/compiler_safe_outputs_job.go` each contained a single orchestration function — `generateMainJobSteps` (658 lines) and `buildConsolidatedSafeOutputsJob` (522 lines) — that had grown to 3× and 2.5× the project's 300-line hard limit respectively. Both functions inlined all phases of their compilation pipeline (setup, checkout, runtime detection, engine installation, agent execution, artifact collection) in one linear body, making individual concerns hard to navigate, test in isolation, or extend without risk of unintended side effects. The Architecture Guardian flagged these as the two highest-priority maintainability debt items in the compiler.

### Decision

We will decompose both functions into thin orchestrators that delegate each distinct compilation phase to a focused helper function. `generateMainJobSteps` becomes a 33-line coordinator calling five helpers (`generateInitialAndCheckoutSteps`, `generateRuntimeAndWorkspaceSetupSteps`, `generateEngineInstallAndPreAgentSteps`, `generateAgentRunSteps`, `generatePostAgentCollectionAndUpload`). `buildConsolidatedSafeOutputsJob` becomes a 43-line coordinator calling three helpers (`buildSafeOutputsSetupAndDownloadSteps`, `buildSafeOutputsHandlerOutputsAndActionSteps`, `buildSafeOutputsJobFromParts`). All helpers remain as methods on `*Compiler` in their respective existing files; no new packages are introduced. External behavior and output structure are preserved identically.

### Alternatives Considered

#### Alternative 1: Retain the Monolithic Functions with Inline Section Comments

Add section comments and godoc cross-references to document the structure of both functions without splitting them. This approach has zero structural diff risk and avoids threading intermediate state across function boundaries. It was rejected because documentation describes structure but does not enforce it — the functions remain single units that cannot be tested or reviewed by concern, and comments drift as the functions evolve. The root problem is co-mingled responsibilities within bodies that are already 3× the project limit, which only decomposition fixes.

#### Alternative 2: Extract Phases into a Dedicated Builder Struct

Introduce a `MainJobBuilder` or `SafeOutputsJobBuilder` struct to group compilation state (e.g., `checkoutMgr`, `engine`, `artifactPaths`) as fields, replacing threaded return values with struct mutation. This would eliminate multi-value returns from phase functions and reduce parameter count. It was not chosen because adding a new type introduces a new abstraction that all future contributors must understand, and the coupling between phases (where outputs of one phase feed directly into the next) is better expressed as explicit function parameters that make data flow visible in the orchestrator. A builder struct would hide this flow behind field assignment, obscuring the sequential dependency.

### Consequences

#### Positive
- Both orchestrators are now short enough to read in full on one screen; the intent and sequencing of each compilation pipeline is visible at a glance.
- Each helper function can be reviewed, tested, and extended independently without risk of touching unrelated phase logic.
- Phase boundaries make the data-flow dependencies between compilation stages explicit in the orchestrator's call sites (e.g., `engine` returned by phase 3 is passed into phases 4 and 5).
- Function-level godoc comments on each helper describe its inputs, outputs, and responsibilities, replacing the inline comment blocks that were previously needed to navigate the monolith.

#### Negative
- Several phase helpers have multi-valued return signatures (e.g., `generateInitialAndCheckoutSteps` returns `(*CheckoutManager, bool, error)`) because intermediate state must be threaded between phases; this increases signature complexity compared to the original inlined local variables.
- The phase boundaries are partially arbitrary and may not align with future feature additions, potentially requiring additional decomposition or consolidation as the compiler grows.

#### Neutral
- All helpers are package-private (`func (c *Compiler) …`), so no public API surface changes.
- Existing integration tests continue to exercise the behavior through the top-level orchestrators; unit tests targeting individual phase helpers can be added incrementally.
- The large diff (141 additions across two files) carries no functional change, which increases reviewer burden to verify behavioral equivalence.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Function Decomposition

1. `generateMainJobSteps` and `buildConsolidatedSafeOutputsJob` **MUST** act as coordinators only: they **MUST NOT** contain inline YAML-emission logic, phase-local variable declarations, or error-generating logic beyond calling the designated phase helpers and threading their return values.
2. Each phase of the main-job and safe-outputs compilation pipelines **MUST** be implemented in a dedicated helper function whose name describes its single responsibility (e.g., `generateAgentRunSteps`, `buildSafeOutputsSetupAndDownloadSteps`).
3. New compilation logic added to either pipeline **MUST** be placed in an appropriately scoped phase helper rather than inlined into the orchestrator.

### File Ownership

1. All main-job phase helpers **MUST** reside in `pkg/workflow/compiler_yaml_main_job.go`; all safe-outputs phase helpers **MUST** reside in `pkg/workflow/compiler_safe_outputs_job.go`, unless a helper is broadly reusable across the `pkg/workflow` package, in which case it **SHOULD** be moved to a shared file within the same package.
2. Phase helpers **MUST NOT** be placed in a separate sub-package unless the sub-package is justified by an independent ADR.

### Function Size

1. Any function in `pkg/workflow/compiler_yaml_main_job.go` or `pkg/workflow/compiler_safe_outputs_job.go` that exceeds 300 lines **MUST** be decomposed before the containing PR can merge.
2. Phase helpers introduced by this ADR **SHOULD** remain under 250 lines to leave headroom for incremental feature additions within each phase.

### Error Handling

1. Errors returned from phase helpers **MUST** be propagated by the orchestrator without additional wrapping unless the orchestrator can provide context that the helper cannot.
2. Phase helpers that emit YAML steps and encounter an error **MUST** return the error immediately without emitting partial output; callers **MUST NOT** use partial results when an error is returned.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25224029071) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
