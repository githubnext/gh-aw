# ADR-48210: Add stringscutprefix Linter for strings.CutPrefix Migration

**Date**: 2026-07-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

Go 1.20 introduced `strings.CutPrefix(s, prefix)` (and the analogous `strings.CutSuffix`) as a cleaner replacement for the two-call idiom `strings.HasPrefix(s, p)` + `strings.TrimPrefix(s, p)`. The two-call pattern is widespread in older Go codebases and carries a silent correctness risk: if the prefix literal or variable is updated in the `HasPrefix` guard but not in the `TrimPrefix` call (or vice versa), the function silently returns the un-trimmed string without any compile-time or runtime error. Many projects, including gh-aw, have accumulated instances of this pattern that were never migrated after `strings.CutPrefix` became available. Manual review alone is insufficient to find and prevent new occurrences across a large codebase.

### Decision

We will implement and register a new custom Go static-analysis linter, `stringscutprefix`, that inspects every `*ast.IfStmt` whose condition is `strings.HasPrefix(s, prefix)` and flags it when the if-body contains a `strings.TrimPrefix(s, prefix)` call with the same arguments (verified by type-object identity for identifiers and by value equality for literals/selector expressions). The linter reports the finding on the `HasPrefix` call site and recommends `strings.CutPrefix`. This linter follows the same structural conventions as the rest of the gh-aw custom linter suite and is registered in `pkg/linters/registry.go`.

### Alternatives Considered

#### Alternative 1: Rely on an Existing Community Linter (e.g., gocritic, revive, or usestdlibvars)

`golangci-lint` bundles several meta-linters (`gocritic`, `revive`, `staticcheck`) that cover a wide range of idioms. If one of them already detected this pattern, a custom linter would be redundant. However, none of the bundled linters enforce the `HasPrefix`+`TrimPrefix` → `CutPrefix` migration at the time of this decision. Using a community linter would eliminate maintenance burden, but the pattern would remain undetected in CI.

#### Alternative 2: Enforce by Coding Guideline Only (No Automated Check)

A written guideline in a style guide or contributing doc would inform developers about `strings.CutPrefix` without requiring tooling changes. This approach has zero implementation cost but provides no enforcement: new occurrences of the anti-pattern would only be caught during code review, inconsistently and long after the code was written. Given the volume of existing violations and the subtle silent-failure mode of the two-call idiom, a documentation-only approach is insufficient.

### Consequences

#### Positive
- CI automatically rejects new instances of the `HasPrefix`+`TrimPrefix` idiom, preventing silent correctness bugs from argument mismatch.
- The linter's argument-identity check (by Go type object) correctly handles both literal strings and variable references, including field selectors, with no regex or text-matching fragility.
- Follows the established gh-aw custom linter pattern (AST traversal via `astutil.Root`, `nolint` directive support, `filecheck` to skip generated files), so the implementation is consistent and immediately reviewable by existing contributors.
- Fixture-based tests via `analysistest` ensure both flagged and non-flagged cases are explicitly enumerated, making the detection boundary clear.

#### Negative
- Adds a new linter package to the registry that the team must maintain going forward (e.g., if the `strings` package API changes or internal `astutil`/`nolint`/`filecheck` helpers are refactored).
- The linter only detects the case where `TrimPrefix` appears directly inside the `IfStmt` body; more complex patterns (e.g., `TrimPrefix` called in a nested helper, or gated on an `else` branch) are not flagged — these require manual review.
- Teams with intentional uses of the two-call idiom (e.g., where the return value of `HasPrefix` is also used for a side effect) must add a `//nolint:stringscutprefix` directive, introducing a small per-site maintenance cost.

#### Neutral
- The linter is appended to the `All()` slice in `registry.go` and inherits the same run-on-all-files behavior as every other registered analyzer; no separate opt-in configuration is required.
- Existing occurrences of the anti-pattern in the codebase are not auto-fixed by this PR; a separate migration sweep would be needed to address historical violations.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
