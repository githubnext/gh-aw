# ADR-54030: Split add_package_manifest.go into Focused Single-Responsibility Files

**Date**: 2026-08-20
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/cli/add_package_manifest.go` had grown to 1330 lines — the largest non-test Go source file in the repository. The file mixed five distinct concerns: top-level orchestration of remote package resolution, YAML manifest parsing and validation, `includes`/`files` field normalization, skill and agent file discovery, and GitHub remote-fetch wrappers. This concentration of concerns in a single file made it hard to navigate, slowed PR reviews (any change touched a large file), and violated the single-responsibility expectation established by similar refactors in the project (see ADR-53818 for the analogous decision on test files).

### Decision

We will split `add_package_manifest.go` into six focused files within the same `pkg/cli` package, each owning a distinct responsibility: shared types and error sentinels (`add_package_manifest.go`), top-level resolve orchestration (`add_package_manifest_resolve.go`), manifest YAML parsing and validation (`add_package_manifest_parse.go`), includes/files field parsing and normalization (`add_package_manifest_includes.go`), skill and agent file discovery (`add_package_manifest_skills.go`), and GitHub remote-fetch wrappers (`add_package_manifest_remote.go`). Code is moved verbatim with no logic changes; per-file import blocks are resolved and doc comments are re-attached where split boundaries separated a comment from its declaration.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file

The simplest option: accept the 1330-line file as a known trade-off. The code compiles and the tests pass. This was rejected because the file was already the largest non-test source in the repo and continued to attract new additions; deferring the split would only increase the cost of a future refactor.

#### Alternative 2: Extract to a dedicated sub-package (`pkg/manifestresolver/`)

Moving the code to its own package would create an enforced API boundary — only exported symbols would be accessible from `pkg/cli`. This was not chosen because the functions are tightly coupled to `pkg/cli` types (`RepoSpec`, `resolvedRepositoryPackage`) and internal helpers, requiring either widespread export of internal types or significant interface indirection. The immediate goal was navigability, not encapsulation; a package boundary can be introduced later if the module grows further.

### Consequences

#### Positive
- Each file has a single, clearly stated responsibility documented in a module-boundary comment at the top.
- PR diffs touching one concern (e.g., parsing) are now scoped to one file, making review easier.
- The split pattern mirrors the existing `add_package_manifest_test.go` / `add_package_manifest_mapping_test.go` organization and the approach documented in ADR-53818.
- Existing tests require no changes and continue to exercise the same package-level behavior.

#### Negative
- Intra-package coupling remains implicit — there is no enforced API boundary between the new files; future authors must rely on naming and doc comments to respect the boundaries.
- Understanding the full end-to-end resolve flow now requires reading across multiple files rather than scrolling within one.

#### Neutral
- No behavior changes: code was moved verbatim; all function signatures, error messages, and test coverage remain identical.
- The `pkg/cli` package's public API surface is unchanged.

---

*ADR created by [adr-writer agent].*
