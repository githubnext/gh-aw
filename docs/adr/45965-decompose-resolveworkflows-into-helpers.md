# ADR-45965: Decompose `ResolveWorkflows` into Focused Helper Functions

**Date**: 2026-07-16
**Status**: Draft
**Deciders**: Unknown (AI-generated from PR diff; authored by copilot-swe-agent)

---

### Context

`ResolveWorkflows` in `pkg/cli/add_workflow_resolution.go` grew into a single function exceeding 250 lines of code. It combined six distinct concerns in one body: input validation, workflow spec parsing (local packages, repository packages, and plain specs), current-repository guard checks, wildcard expansion, per-spec content fetching and metadata extraction, and bootstrap profile selection. The `largefunc` linter flagged this as a maintainability problem. Functions of this size are difficult to read, review, and test in isolation. The codebase uses the lint-enforced `largefunc` policy as a code-quality gate, so the violation needed to be resolved.

### Decision

We will decompose `ResolveWorkflows` into a pipeline of single-responsibility internal helper functions, keeping the public signature and external call sites unchanged. Each phase (validation, parsing, guard-checking, wildcard expansion, resolution, bootstrap selection) becomes its own named function, with an intermediate `specResolutionResult` struct carrying state between phases.

### Alternatives Considered

#### Alternative 1: Maintain the Monolithic Function

Accept the `largefunc` lint finding and disable or suppress the lint rule for this function. This avoids any risk of behavioral regression from restructuring.

Not chosen because the project uses `largefunc` as an active quality gate; suppressing it incurs ongoing lint debt and leaves the function difficult to test and review.

#### Alternative 2: Struct-Based Resolver with Methods

Introduce a `workflowResolver` struct and convert each phase into a method, using fields to share state between phases instead of a pass-through result struct.

Not chosen because it would require changing the call site signature or wrapping the public function in a factory, adding abstraction overhead. For a single entrypoint that is not instantiated in different configurations, a free-function pipeline is simpler and the struct carries no meaningful object lifetime.

### Consequences

#### Positive
- Resolves one `largefunc` lint violation in `pkg/cli`, keeping the lint gate clean.
- Each helper function has a clear, testable boundary — input validation, spec parsing, wildcard expansion, and bootstrap selection can each be unit-tested independently.
- The `ResolveWorkflows` orchestrator reads as a high-level pipeline, making control flow auditable without tracing a 250-line monolith.
- `specResolutionResult` makes intermediate state explicit and typed instead of implicitly carried in three parallel local variables.

#### Negative
- More function call sites across the file increase the surface area to navigate when debugging end-to-end behavior.
- The five-return-value signatures on dispatch helpers (`specs, warnings, bootstrapProfile, handled, error`) are non-idiomatic Go and may be confusing to new contributors.
- Each new helper is unexported; if future tests need finer-grained coverage they must be in the same package, which is already the case but limits test organization options.

#### Neutral
- Public interface and all external call sites are unchanged; no callers need to be updated.
- The refactor is strictly a code organization change — no behavioral changes to validation logic, error messages, or warning aggregation are introduced.
- The `specResolutionResult` struct is local to the file; it is not part of the exported API.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
