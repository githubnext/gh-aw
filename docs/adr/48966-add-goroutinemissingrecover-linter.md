# ADR-48966: Add goroutinemissingrecover Linter to Enforce Goroutine Panic Recovery

**Date**: 2026-07-29
**Status**: Draft
**Deciders**: Unknown

---

### Context

Goroutines started via function literals (`go func() { ... }()`) that panic will terminate the entire process; unlike panics in the calling goroutine, they cannot be caught by the caller's `recover`. The codebase already has examples of the safe pattern (`pkg/console/spinner.go`, `pkg/cli/docker_images.go`) but the pattern was applied inconsistently — `pkg/cli/forecast_compute.go` launched worker goroutines with no panic protection. Manual code review had not caught this divergence. The existing `pkg/linters/` framework provides the infrastructure for adding custom `go/analysis` passes that are automatically applied across the entire codebase at review time.

### Decision

We will add a new `goroutinemissingrecover` custom `go/analysis` linter under `pkg/linters/goroutinemissingrecover/` that flags any goroutine started via a function literal whose body does not install a top-level `defer func() { recover() }()` guard. The linter uses the existing `nolint.HasDirectiveForLinter` mechanism so call sites that intentionally skip recovery can be explicitly documented with `//nolint:goroutinemissingrecover`. Named-function goroutines (`go f()`) are out of scope because the named function can install its own recovery.

### Alternatives Considered

#### Alternative 1: Rely on Code Review (Manual Enforcement)

Code reviewers catch missing recover guards in goroutine function literals as part of the normal PR review process. This was already the implicit policy; the inconsistency between `forecast_compute.go` and `spinner.go`/`docker_images.go` demonstrates it is insufficient at scale. As the codebase grows, reviewers cannot reliably spot every unguarded goroutine literal across hundreds of files. Manual enforcement does not scale and produces no audit trail.

#### Alternative 2: Use a Third-Party Linter (e.g., `gocritic` or `revive`)

Add a general-purpose linter that has a goroutine-recovery rule, rather than building a bespoke `go/analysis` pass. This avoids new code but introduces a new external dependency for a single semantic rule. Existing third-party rules in this space do not precisely match the required semantics (function-literal goroutines only, integration with the project's `nolint` index and `filecheck` generated-file skip logic). Adapting a third-party rule would require the same investigation effort as writing the custom pass, with less control over the result.

### Consequences

#### Positive
- Unguarded goroutine function literals are caught at CI time rather than at runtime, preventing process-killing panics from reaching production.
- The `nolint:goroutinemissingrecover` suppression mechanism creates an explicit, searchable record of every intentional exception to the rule.
- The linter follows the conventions of existing passes (`astutil.Inspector`, `nolint`, `filecheck`) and is registered automatically via `linters.All()`, requiring no changes to the runner infrastructure.

#### Negative
- Every new goroutine function literal added to the codebase must include a boilerplate `defer func() { if r := recover(); r != nil { ... } }()` block or an explicit `nolint` directive, increasing the per-goroutine authoring cost.
- Named-function goroutines (`go f()`) are intentionally out of scope; a developer who refactors a function literal into a named function to avoid the linter may not actually add recovery to the named function, shifting rather than solving the problem.

#### Neutral
- The analyzer count in `pkg/linters/doc.go` and `pkg/linters/README.md` increments from 59 to 60; the `spec_test.go` documented-analyzer list must be kept in sync whenever a new analyzer is added.
- Generated files are skipped via `filecheck.ShouldSkipFilename`, consistent with all other custom analyzers in this package.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
