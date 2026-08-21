# ADR-54611: Centralize gh CLI Error Text Classification in errorutil

**Date**: 2026-08-21
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

Three sites in `pkg/cli` used inline `strings.Contains(err.Error(), "...")` calls to classify `gh` CLI error conditions (`INSUFFICIENT_SCOPES`, `already merged`, `MERGED`). These scattered checks are brittle — if the CLI's error text changes, all sites silently break. Only one of the three sites was documented with a `//nolint:errstringmatch` justification; the other two were unflagged and inconsistent with the project's convention. The `errstringmatch` linter enforces that raw error-string matching must be suppressed explicitly, creating accumulating lint debt at every new call site.

### Decision

We will add `IsInsufficientScopesError(err error) bool` and `IsAlreadyMergedError(err error) bool` helpers to `pkg/errorutil`, mirroring the existing `IsNotFoundError`/`IsForbiddenError` pattern. The error-text matching is centralized inside `errorutil`, so call sites in `pkg/cli` use named helper calls instead of raw string matching. This eliminates per-site `nolint` comments and consolidates the fragile literals in one reviewed, documented, and tested location.

Each helper keeps the narrowest match that still covers the known gh CLI wording:

- `IsInsufficientScopesError` matches the `INSUFFICIENT_SCOPES` GraphQL error type case-insensitively; the literal is specific enough that broader wording cannot collide with it.
- `IsAlreadyMergedError` matches the case-insensitive phrase `already merged` or the case-sensitive GraphQL state literal `MERGED`. Matching the bare lowercase word `merged` was explicitly rejected because failure wording such as "could not be merged" or "not merged" would then be classified as a successful merge, causing `handleMergeAttempt` to report success and stop retrying after a failed merge. Negative tests pin this behavior.

### Alternatives Considered

#### Alternative 1: Keep Inline string Checks with Per-Site nolint Annotations

Leave the `strings.Contains(err.Error(), ...)` calls in place and add `//nolint:errstringmatch` suppressions to all unflagged sites. This requires no new API surface but duplicates the brittle string literals across call sites, making it hard to update them consistently if the CLI's error text ever changes. It also accumulates inconsistent lint suppressions without a single place to review or update the classification logic.

#### Alternative 2: Structured Errors from the gh CLI Wrapper Layer

Instead of matching error text, expose structured sentinel errors or typed errors from the `workflow` package that wraps `gh` CLI calls, eliminating string matching entirely. This is the most robust approach but requires significant changes to the `workflow` package and the `gh` CLI wrapper layer — changes that are out of scope for this targeted refactor. The `gh` CLI does not natively return structured errors for these conditions, so this would also require parsing or wrapping CLI output at the source.

### Consequences

#### Positive
- Brittle error-text substring matches are consolidated in one reviewed location — future text changes require only a single update in `pkg/errorutil`.
- Call sites in `pkg/cli` are simplified and linter-clean without per-site `nolint` suppressions.
- The new helpers follow the established `IsNotFoundError`/`IsForbiddenError` convention, reducing cognitive load for future contributors.

#### Negative
- String matching is still used under the hood — the approach centralizes fragility rather than eliminating it; a `gh` CLI text change still silently breaks behavior, just at fewer, more visible locations.
- The `pkg/errorutil` package API surface grows with each new error category; without governance, this could accumulate many narrowly-scoped helpers over time.

#### Neutral
- Matching semantics are deliberately asymmetric (case-insensitive phrase plus case-sensitive state literal), which is documented in the helper doc comments, `pkg/errorutil/README.md`, and pinned by negative tests.
- The `pkg/cli/update_extension_check.go` `isWindowsLockError` function was intentionally excluded — it matches local Windows-specific stdout/stderr text that is not a structured GitHub/gh CLI error category and already carries a proper `nolint` justification.
- Unit tests and spec tests were added for both new helpers following the existing test pattern in `pkg/errorutil`.

---

*ADR created by [adr-writer agent] and finalized to reflect the classification semantics implemented in this PR.*
