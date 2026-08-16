# ADR-53018: Centralize GitHub Error Classifiers and Value Formatting

**Date**: 2026-08-16
**Status**: Draft
**Deciders**: Unknown

---

### Context

The codebase had duplicate implementations of two cross-cutting concerns scattered across multiple packages. `IsAuthError` and `IsRateLimitError` were defined in `pkg/gitutil` but called by `pkg/cli` and `pkg/parser` — packages that have no semantic dependency on git operations. Placing error-classification logic in a git utility package created an inappropriate coupling: callers that only needed to classify GitHub API responses had to import git infrastructure. Independently, `marshalEnvValue` in `pkg/workflow` contained inlined JSON/reflect normalization that duplicated the same logic already present in `importinpututil.FormatResolvedValue`, creating a split-brain risk where the two serialization paths could diverge silently. Additionally, full-SHA validation was written inline as `len(x)==40 && gitutil.IsHexString(x)` at seven separate call sites rather than using the already-exported `gitutil.IsValidFullSHA` predicate.

### Decision

We will move `IsAuthError` and `IsRateLimitError` out of `pkg/gitutil` and into `pkg/errorutil` as the canonical shared API for GitHub error classification. We will update all callers across `pkg/cli` and `pkg/parser` to import from `errorutil`. We will replace `marshalEnvValue`'s inlined JSON/reflect normalization with a delegation to `importinpututil.FormatResolvedValue`, keeping only a `fmt.Sprint` scalar fallback and a `nil`→`""` guard. We will replace all inline `len(x)==40 && IsHexString(x)` predicates with `gitutil.IsValidFullSHA`.

### Alternatives Considered

#### Alternative 1: Keep classifiers in `gitutil`, add re-export shims in `errorutil`

Re-export `gitutil.IsAuthError` and `gitutil.IsRateLimitError` from `errorutil` without moving the implementation. Callers can import from either package. This avoids touching the implementation and keeps `gitutil` as the authority, but it creates two public APIs for the same function, does not fix the semantic mismatch (error classification is not a git concern), and leaves the underlying coupling intact. It was rejected because it trades a clean break for ongoing confusion about which package owns the behavior.

#### Alternative 2: Inline error-classification logic at each call site

Remove shared classifiers entirely and duplicate the substring checks wherever they are needed. This eliminates the package-dependency question but defeats the goal of a single source of truth, making future changes to classification phrases error-prone and requiring updates across many files. It was rejected because the problem that motivated `gitutil.IsAuthError` in the first place — avoiding scattered inline checks — would recur immediately.

### Consequences

#### Positive
- `pkg/gitutil` scope is now narrowly defined as git repository operations and SHA/ref validation, eliminating an inappropriate coupling to GitHub API error semantics.
- `pkg/errorutil` becomes the single authoritative location for GitHub error classification, so future phrase changes need to be made in exactly one place.
- `marshalEnvValue` and `importinpututil.FormatResolvedValue` are guaranteed to produce identical serialization for arrays and maps, eliminating the risk of silent divergence between the two code paths.
- Inline SHA predicates are replaced by a named, tested, regex-backed function, reducing the chance of off-by-one errors (e.g., accepting mixed-case or 64-character SHAs).

#### Negative
- The change touches 21 files across `pkg/cli`, `pkg/parser`, `pkg/workflow`, `pkg/gitutil`, and `pkg/errorutil`, making it a wide-surface refactor that carries merge-conflict risk for any concurrent branches importing `gitutil.IsAuthError`.
- Removing `IsRateLimitError` and `IsAuthError` from `pkg/gitutil`'s public API is a breaking change for any external consumers that imported those symbols directly (though this appears to be an internal-only codebase).

#### Neutral
- `isPermissionErrorStr` in `pkg/cli/audit.go` now delegates to `errorutil.IsAuthError` and augments with audit-specific markers (`exit status 4`, `permission`, `gh auth login`, workflow guidance) rather than maintaining its own canonical union — this preserves audit-command-specific behavior without duplicating shared logic.
- Tests for the moved functions are migrated from `pkg/gitutil` to `pkg/errorutil`, and spec tests are updated to reflect the new package ownership.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
