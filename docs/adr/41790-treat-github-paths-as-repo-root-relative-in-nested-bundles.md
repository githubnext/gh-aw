# ADR-41790: Treat `.github/` Paths as Repo-Root-Relative in Nested Bundle Manifests

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown (Copilot-authored fix; see PR #41790)

---

### Context

The gh-aw tooling supports "nested bundles" — agentic workflow manifests (`aw.yml`) stored at a sub-path within a repository (e.g., `dependabot/aw.yml`). Bundle manifests can enumerate workflow files under a `files:` key, including paths under `.github/workflows/`. Prior to this fix, the path normalization functions `normalizePackageInstallablePaths` and `scanRepositoryPackageInstallablePaths` unconditionally prepended the bundle's package path to every entry, causing `.github/workflows/foo.md` to resolve to `dependabot/.github/workflows/foo.md` — a path that does not exist, producing HTTP 404 errors at install time. The `.github/` directory is a GitHub-defined repository-root convention; no sub-path equivalent exists.

### Decision

We will treat any path beginning with `.github/` in a bundle manifest as repo-root-relative, bypassing package-prefix logic entirely. All other supported path prefixes (`workflows/`, `agentic-workflows/`) remain relative to the bundle's package root and continue to receive the package-path prefix. This convention is encoded in both `normalizePackageInstallablePaths` (explicit `files:` entries) and `scanRepositoryPackageInstallablePaths` (auto-scan), so the rule applies uniformly regardless of how paths reach the resolver.

### Alternatives Considered

#### Alternative 1: Require manifest authors to always use full repo-root paths

Manifest authors could be required to write `.github/workflows/foo.md` without any special handling — the same path would be expected from both root-level and nested bundles, and the normalization code would be left unchanged. This was rejected because the current normalization logic would silently corrupt any such path by prepending the package directory, and because fixing that would require all existing nested manifests to be audited and updated, with no compiler-enforced guarantee of correctness.

#### Alternative 2: Refactor path representation to store full repo-root paths universally

A deeper alternative would be to change the internal data model so that all collected paths are stored as full repo-root-relative paths from the moment they are collected, removing the dual representation entirely. `isSupportedPackageInstallablePath` would be updated to validate against full paths. This was rejected as out-of-scope for a targeted bug fix: it would require coordinated changes across multiple callers and data structures, and carries a higher risk of introducing regressions in root-bundle behaviour, which is unaffected by the current bug.

### Consequences

#### Positive
- Nested bundles that reference `.github/workflows/` paths now install correctly without HTTP 404 errors.
- The convention mirrors the behaviour of GitHub itself — `.github/` is always a repo-root directory — reducing cognitive overhead for manifest authors.
- Auto-scan for nested bundles now correctly strips the package prefix before validating path support, so previously silently-dropped scan results surface as expected.

#### Negative
- The path resolution semantics are now context-sensitive: the same path string is interpreted differently depending on its prefix (`.github/` vs everything else). This implicit rule is documented only in code comments, which may not be visible to future manifest authors or contributors unfamiliar with the area.
- There is no compile-time or schema-level enforcement of the convention; a future refactor of path normalization could inadvertently break the special case if the comments are not noticed.

#### Neutral
- The fix requires no changes to the `aw.yml` schema or any public-facing API surface; existing valid manifests are unaffected.
- New tests (`TestResolveWorkflows_NestedRepositoryPackage_GithubWorkflowsPathIsRepoRoot`, `TestResolveWorkflows_NestedRepositoryPackage_AutoScan`) encode the expected behaviour and will catch regressions in both paths.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
