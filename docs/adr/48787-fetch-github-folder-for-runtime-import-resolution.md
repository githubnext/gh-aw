# ADR-48787: Fetch Entire `.github/` Folder for Runtime-Import Dependency Resolution

**Date**: 2026-07-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw trial` command installs a workflow in trial mode by fetching its declared dependencies: frontmatter `imports:`, `@include` directives, dispatch/call workflows, and resources. However, workflows may also reference files via `{{#runtime-import .github/path/to/file}}` macros in the workflow body. These macro-resolved paths were never traversed by the explicit dependency scanner, so any workflow using `runtime-import` would fail during template interpolation with "Runtime import file not found." The root cause is that `.github/`-local files are an implicit, untracked dependency class with no manifest.

### Decision

We will perform a sparse checkout of the entire `.github/` directory from the source repository at the workflow's exact ref before calling `fetchAllRemoteDependencies`. The sparse-checked-out directory is merged into the trial host using a non-destructive copy that skips existing files, preserving the already-injected trial workflow. The `.github/` copy step is non-fatal: if it fails, trial install falls through to the existing individual dependency fetching path.

### Alternatives Considered

#### Alternative 1: Extend the dependency scanner to parse `{{#runtime-import}}` macros

The dependency scanner could be taught to recognize `{{#runtime-import ...}}` syntax and add the resolved paths to the explicit fetch list handled by `fetchAllRemoteDependencies`. This would be surgical — only the referenced files are fetched — but requires maintaining a parser for macro syntax. It would also miss any future macro or implicit reference pattern not yet in the scanner, meaning the fix would be fragile against new macro types. The current diff shows no attempt to extend the scanner, indicating this was considered and rejected in favor of a more complete solution.

#### Alternative 2: Fail fast and require authors to declare all `runtime-import` files as explicit dependencies

Trial mode could require workflow authors to also list `runtime-import` targets in the frontmatter `imports:` field. This places the burden on the author, breaks existing workflows that already use the macro without redundant declarations, and would make the UX inconsistent (why does `@include` work automatically but `runtime-import` doesn't?). The bug report (issue #47147) confirms this degraded experience is the problem being fixed.

### Consequences

#### Positive
- All `.github/`-local resources — including `runtime-import` files, skills, and any future implicit dependency patterns — are automatically available in the trial host without requiring workflow authors to redeclare them.
- The non-fatal fallback preserves the existing individual dependency fetch path, so trial install degrades gracefully if the sparse checkout fails (e.g., network issue or unsupported host).
- Path traversal is guarded via `fileutil.ValidatePathWithinBase`, preventing symlink or `..`-escape attacks during the merge.

#### Negative
- Every non-local trial activation now performs an additional git sparse checkout, increasing network I/O even for workflows that do not use `runtime-import`. Repositories with large `.github/` directories will incur non-trivial latency.
- The sparse checkout creates and immediately removes a temporary directory per activation, adding filesystem churn.

#### Neutral
- `fetchAllRemoteDependencies` is retained alongside the new step: cross-repo `@include`, dispatch, and resource dependencies still require individual fetch logic, so the two mechanisms coexist rather than one replacing the other.
- The `mergeDirectory` skip-existing semantics mean that future `.github/` changes at the ref are not visible once the trial-modified workflow has been written, which is intentional but may surprise debuggers.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
