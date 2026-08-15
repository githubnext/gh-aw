# ADR-52218: Extract Helper Functions to Reduce pkg/cli largefunc Backlog

**Date**: 2026-08-13
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

A daily `make golint-custom` run surfaces `largefunc` violations in `pkg/cli/add_package_manifest.go`. Three functions exceeded the line-length threshold: `resolveRepositoryPackage`, `parseRepositoryPackageManifest`, and `resolvePackageSkillFiles`. Each function mixed multiple responsibilities (slug parsing, ref resolution, YAML parsing, version checking, metadata population, skill-directory scanning) in a single body, making individual steps hard to test or reason about in isolation. Issue #52208 tracks the full `pkg/cli` largefunc backlog; this PR addresses the first slice.

### Decision

We will decompose the three oversized functions into focused, single-responsibility helper functions using the Extract Function refactoring pattern. The helpers remain package-level functions in `pkg/cli/add_package_manifest.go` and use consistent `RepositoryPackage`-oriented naming (for example `resolveRepositoryPackage*`, `parseRepositoryPackage*`, and `populateRepositoryPackageManifest*`) to stay self-documenting. No observable behavior changes; the refactoring is a pure structural improvement to eliminate lint violations.

### Alternatives Considered

#### Alternative 1: Suppress lint warnings with `//nolint:largefunc` directives

Add per-function suppression comments and leave the functions as-is. This is the lowest-effort option. It was rejected because silencing the linter removes the signal without fixing the underlying maintainability concern; the functions remain difficult to read and individually test.

#### Alternative 2: Reorganize as method receivers on a new struct

Introduce a `repositoryPackageResolver` struct and convert the pipeline into chained method calls (e.g., `r.splitSlug()`, `r.resolveRef()`, …). This would co-locate state and reduce parameter threading. It was not chosen for this PR because it represents a larger semantic restructuring that goes beyond the targeted lint-reduction scope; the struct shape would need agreement across the team before adoption, and the issue specifically asks for minimal helper extractions.

### Consequences

#### Positive
- `largefunc` lint violations in `add_package_manifest.go` are eliminated, keeping the `make golint-custom` baseline clean.
- Each extracted helper has a single, named responsibility and can be tested and reasoned about independently.
- The extracted helper functions keep callsites readable while avoiding deeply nested logic.

#### Negative
- The `pkg/cli` package namespace grows with several new `resolveRepositoryPackage*` and `parseRepositoryPackageManifest*` top-level functions, which can feel cluttered when browsing the file.
- The newly introduced `repositoryPackageExtensionFiles` result type is an additional concept callers must learn, even though its scope is intentionally local.

#### Neutral
- All existing behavior is preserved; this is a zero-semantic-change refactoring.
- The extracted-function pattern established here will be replicated by follow-on PRs that address the remaining `pkg/cli` largefunc findings tracked in issue #52208.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
