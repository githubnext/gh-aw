# ADR-52982: Options Structs for Parameter-Heavy Functions

**Date**: 2026-08-15
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Two functions in the `pkg/workflow` and `pkg/cli` packages accumulated more parameters than the project's custom linter permits. `EnforceSafeUpdate` had six positional parameters (manifest, secret names, action refs, redirect string, PR transition, memory-validation scripts), and `resolveRepositoryPackageExtensionFiles` had nine (context plus eight domain values). The custom parameter-count linter flagged both as violations in CI, requiring a fix before the branch could merge.

Positional parameter lists of this length are brittle: callers must supply all arguments in the correct order, zero values (empty string, nil slice) are indistinguishable from intentional omissions, and adding a new parameter requires updating every call site.

### Decision

We will consolidate the excess parameters of each function into a dedicated options struct (`SafeUpdateOptions` for `EnforceSafeUpdate`, and `repositoryPackageExtensionFilesOptions` for `resolveRepositoryPackageExtensionFiles`). Each function now accepts a context (where applicable) and a single options value. All existing call sites are updated to use named struct-literal syntax. The security semantics and internal logic of both functions remain unchanged.

### Alternatives Considered

#### Alternative 1: Raise or disable the linter parameter-count threshold

The linter threshold could be increased or the specific functions could be annotated to suppress the check. This would silence the CI failure without changing the API surface.

Not chosen because it defeats the purpose of the lint rule: long positional parameter lists remain a maintenance liability regardless of whether the linter ignores them, and relaxing the threshold for individual functions normalizes an anti-pattern that is already causing issues.

#### Alternative 2: Decompose functions into smaller units

`EnforceSafeUpdate` could be split into separate functions for secret enforcement, action enforcement, redirect enforcement, and memory-script enforcement, each taking only the parameters it needs.

Not chosen because the six checks are tightly coupled — they share the same manifest baseline and must all pass before a safe-update review can be approved. Splitting them would require callers to coordinate multiple calls and aggregate errors, increasing complexity at every call site without a clear architectural gain.

### Consequences

#### Positive
- Function signatures are stable: adding a new enforcement input only requires a new field on the options struct; existing call sites compile unchanged.
- Named fields at call sites make arguments self-documenting and eliminate ordering errors.
- Zero-value fields express deliberate omission clearly (e.g., `nil` Manifest conveys "no lock file" by struct default).

#### Negative
- Struct-literal call sites are more verbose than the previous positional style, especially for callers that pass most fields.
- The options struct is a public type (`SafeUpdateOptions`), so it becomes part of the package API surface; future field additions are additive but field removals are breaking changes.

#### Neutral
- Test files required mechanical updates to switch from positional arguments to named struct fields; test coverage and assertions are unchanged.
- The `repositoryPackageExtensionFilesOptions` struct is unexported, so its scope is limited to the `cli` package and carries no external API obligations.

---

*ADR created by [adr-writer agent]. Finalized and accepted as part of PR #52982.*
