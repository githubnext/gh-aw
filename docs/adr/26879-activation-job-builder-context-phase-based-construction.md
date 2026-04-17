# ADR-26879: Activation Job Builder Context for Phase-Based Construction

**Date**: 2026-04-17
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/workflow/compiler_activation_job.go` grew to 1084 lines with a single `buildActivationJob`
function of 624 lines, triggering Architecture Guardian violations for both file-size and
function-size thresholds. The function accumulated 15+ local variables (steps, outputs, reaction
flags, label-command state, app-token flags) that were threaded through dozens of inline logic
blocks. A straightforward extraction into private helpers would require passing that growing set
of parameters between each helper, trading one form of complexity for another.

### Decision

We will introduce a dedicated `activationJobBuildContext` struct to carry all mutable construction
state across focused builder phases. The orchestrator (`buildActivationJob`) becomes a thin
sequencer that calls phase methods such as `addActivationFeedbackAndValidationSteps` and
`addActivationRepositoryAndOutputSteps`; each phase reads from and mutates the shared context.
Builder logic is co-located in a new file, `compiler_activation_job_builder.go`, while the
orchestrator remains in `compiler_activation_job.go`. This keeps parameter lists minimal and
makes each construction phase independently readable.

### Alternatives Considered

#### Alternative 1: Parameter Threading

Extract helper functions that receive individual parameters for each piece of state they need.
This avoids a new struct type and keeps each helper's dependencies explicit in its signature.
It was rejected because the activation job manages 15+ distinct state fields; threading them
through every helper would produce large, fragile parameter lists and obscure which helpers
modify which state.

#### Alternative 2: Immutable Fluent Builder

Adopt a fluent builder pattern where each phase returns a new builder value (immutable-style),
offering clearer data-flow audit trails. It was not chosen because activation-job construction
is always sequential and single-call; the overhead of immutable value copying adds complexity
without a practical benefit in this synchronous, single-goroutine context.

#### Alternative 3: Leave the Monolithic Function Intact

Keep `buildActivationJob` as a single 624-line function and suppress or accept the Architecture
Guardian violations. This was rejected because the violations represent genuine readability and
maintainability problems, not false positives, and deferring the split would make future
extensions progressively harder.

### Consequences

#### Positive
- `buildActivationJob` shrinks from 624 lines to ~40 lines, making the construction sequence immediately legible.
- `compiler_activation_job.go` is reduced from 1084 to 497 lines, resolving the file-size violation.
- Each builder phase is independently readable and has a well-defined mutation contract documented via Go doc comments.
- Error propagation is centralised at the orchestration boundary with a consistent `fmt.Errorf("...: %w", err)` convention.

#### Negative
- `activationJobBuildContext` is a mutable, order-dependent struct; phases must be invoked in the correct sequence or the context can be left in an inconsistent state.
- The new builder file (`compiler_activation_job_builder.go`) is 473 lines and may itself approach size limits if activation logic continues to grow.
- Unit-testing individual phases requires constructing a populated `activationJobBuildContext`, increasing test-setup boilerplate slightly.

#### Neutral
- Both files remain in the same `workflow` package; build constraints and imports are unchanged.
- `activationJobBuildContext` is unexported, so this is a purely internal refactor with no public API surface change.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**,
> **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be
> interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Build Context Lifecycle

1. Implementations **MUST** create a new `activationJobBuildContext` via `newActivationJobBuildContext` before invoking any builder phase method.
2. Implementations **MUST NOT** share a single `activationJobBuildContext` instance across multiple concurrent `buildActivationJob` invocations.
3. The `activationJobBuildContext` struct **MUST** remain unexported and **MUST NOT** appear in any public API surface.
4. Implementations **MUST NOT** read fields of `activationJobBuildContext` before they are populated by the appropriate initialisation or phase method.

### Phase Sequencing

1. Builder phase methods **MUST** be invoked in the order defined in `buildActivationJob`: feedback/validation → repository/outputs → command/label outputs → needs/condition → prompt generation → artifact upload.
2. Phase methods that can return an error **MUST** have their errors propagated; errors **MUST NOT** be silently ignored.
3. New builder phases **SHOULD** be added as dedicated `*Compiler` methods rather than inlined into `buildActivationJob`.
4. Each phase method **SHOULD** include a Go doc comment describing which `activationJobBuildContext` fields it reads and which it mutates.

### File Organisation

1. The orchestrator function `buildActivationJob` **MUST** remain in `compiler_activation_job.go`.
2. The `activationJobBuildContext` type definition and all phase methods **MUST** reside in `compiler_activation_job_builder.go`.
3. Both files **MUST** remain in the `workflow` package.
4. Either file **SHOULD NOT** exceed 600 lines; if a file approaches this limit, phase logic **SHOULD** be extracted into additional focused files within the same package.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and
**MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement
constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24573630987) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
