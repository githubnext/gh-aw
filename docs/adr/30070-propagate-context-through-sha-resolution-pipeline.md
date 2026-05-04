# ADR-30070: Propagate context.Context Through Action SHA Resolution and Compilation Pipeline

**Date**: 2026-05-04
**Status**: Draft
**Deciders**: Unknown (AI-generated draft — review and finalize before accepting)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh-aw` CLI resolves GitHub Actions SHA pins by calling GitHub's REST API at multiple points during workflow compilation and the `add` command pipeline. These calls were previously made using hardcoded `context.Background()` contexts, which made every API call immune to cancellation and timeouts. When GitHub's API is slow or unreachable, the tool would hang indefinitely with no way for the caller — including Cobra command handlers, tests, or programmatic users — to interrupt the operation. Go's standard convention for cancellable I/O is to accept a `context.Context` as the first parameter and pass it through to all downstream I/O calls.

### Decision

We will thread `context.Context` as the first parameter through every function in the action SHA resolution and workflow compilation call stacks that performs or delegates I/O, replacing all internal uses of `context.Background()` with the caller-supplied context. Where threading context through a method signature is impractical (specifically for `Compiler` internal methods called from many dispatch sites), we store the context on the `Compiler` struct via `SetContext(ctx)` with a `context.Background()` fallback, making the intent visible while preserving backward compatibility for call sites that have no available context.

### Alternatives Considered

#### Alternative 1: Keep Hardcoded `context.Background()` (Status Quo)

Every I/O call would continue to use `context.Background()`, ignoring any cancellation signal from the caller. This is simple and requires no API changes, but it means timeouts and context cancellations from Cobra (e.g., Ctrl-C) have no effect on in-flight GitHub API calls, causing potential indefinite hangs in CI and watch-mode scenarios.

#### Alternative 2: Package-Level or Global Context Variable

Store the "current" context in a package-level variable, updated at command entry points. This avoids changing dozens of function signatures but is not idiomatic Go, introduces implicit state shared across goroutines (data-race risk), and makes testing harder since tests cannot inject isolated contexts. This approach was rejected because the signature-threading approach, while verbose, is the standard Go pattern and is safe under concurrent test execution.

#### Alternative 3: Full Context on `Compiler` Struct (Struct-Only Storage)

Store context exclusively on the `Compiler` struct and access it through a method everywhere, avoiding parameter changes entirely. This would be more consistent but would hide context flow from function signatures, making it harder for readers to see which functions perform I/O. The hybrid approach chosen — threading context via parameters for most functions and falling back to struct storage only for compiler-internal methods — keeps the call graph readable for the common case.

### Consequences

#### Positive
- Callers (Cobra commands, integration tests, background agents) can now cancel or time out all GitHub API calls by cancelling the context they supply.
- `TestCheckActionSHAUpdates_ContextCancellation` demonstrates and enforces the new behavior: a pre-cancelled context causes resolution to skip cleanly rather than hang.
- The code is now conformant with Go's standard context propagation idiom throughout the I/O paths.

#### Negative
- Every function in the changed call stack has a new leading `context.Context` parameter, which is a breaking API change for any external callers of the affected public functions (`AddResolvedWorkflows`, `CompileWorkflowWithValidation`, `CompileWorkflowDataWithValidation`, `CheckActionSHAUpdates`, `ValidateActionSHAsInLockFile`).
- The `Compiler` struct now carries mutable state (the stored context), which can be surprising in concurrent uses of a single `Compiler` instance.
- Call sites that have no meaningful context (e.g., `enable.go` watch-mode compilation) must explicitly pass `context.Background()`, adding boilerplate without behavioral change.

#### Neutral
- `FetchDefaultBranch` switches from `RunGH` to `RunGHContext` as a consequence of receiving a real context; callers see no change in behavior when the context is not cancelled.
- The `Compiler.context()` accessor falls back to `context.Background()` if `SetContext` was never called, preserving existing behavior for callers that have not been updated yet.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Context Propagation in I/O Functions

1. Any function that performs or delegates a GitHub API call, git remote operation, or other network I/O **MUST** accept a `context.Context` as its first parameter.
2. Functions **MUST NOT** replace a caller-supplied context with `context.Background()` before passing it to a downstream I/O call unless the function is explicitly documented as a fire-and-forget background operation.
3. Callers that have no meaningful context available (e.g., watch-mode loops, legacy wrappers) **SHOULD** pass `context.Background()` explicitly and document why they cannot propagate a richer context.
4. New internal helpers that perform only pure computation (no I/O) **MAY** omit the context parameter.

### Compiler Context Storage

1. When threading context through every method signature of `Compiler` is impractical, implementations **MUST** call `compiler.SetContext(ctx)` before invoking compilation so that compiler-internal I/O inherits the caller's context.
2. `Compiler.context()` **MUST** return `context.Background()` as a fallback when no context has been set via `SetContext`, ensuring the `Compiler` is safe to use without explicit context injection.
3. `Compiler` instances **SHOULD NOT** be shared across goroutines after `SetContext` has been called, because the stored context field is mutable and not protected by a mutex.

### Testing

1. Tests for functions that accept a context **MUST** pass a real (non-nil) context, and **SHOULD** include at least one test case that verifies behavior under a pre-cancelled context.
2. Tests **MUST NOT** pass `nil` as a context argument; they **MUST** pass at minimum `context.Background()`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25323021162) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
