# ADR-54417: Force-Refresh Package-Managed Assets and Unify Frontmatter Ref Updates During `gh aw update`

**Date**: 2026-08-21
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

`gh aw update` is responsible for keeping a repository's AI workflow configuration in sync with upstream packages and registered skill/plugin sources. Prior to this change, the manifest reconciliation step used add-only semantics: if a destination file already existed on disk, the reconciler logged a skip message and continued without re-downloading the upstream version. Similarly, the frontmatter ref-update pipeline rewrote `skills` entries but left `plugins` entries untouched. The net effect was that running `gh aw update` after an upstream package bumped its content produced no local change — the lock file advanced, but the actual workflow, skill, and agent files remained at their stale versions. Users were forced to manually delete managed files before running `update` to trigger a re-download.

### Decision

We will change `reconcileManifestManagedAssets` to overwrite existing package-managed assets — action workflows, skill files, and agent files — instead of skipping them, removing the exist-check short-circuit that previously caused skips. Overwriting is gated by package ownership: a destination is only refreshed automatically when it is tracked as owned by the current package (via `.github/aw/packages` ownership records) and has not drifted from its installed state, or when the caller passes `--force`. Destinations that are not tracked as owned by the package, or that have local modifications, cause `update` to fail with an explicit error rather than silently overwriting an unrelated or locally-modified file. Concurrently, we extract a shared `updateFrontmatterRepoRefsInContentWithResolver` function that handles both `skills` and `plugins` frontmatter entries via a `fieldName`/`objectKey` parameter pair, so that plugin SHA refs are updated by the same resolver path used for skills. This gives `gh aw update` consistent, reliable "sync to upstream" semantics across all asset types while still protecting unrelated or locally-modified files from being clobbered.

### Alternatives Considered

#### Alternative 1: Add an Opt-In `--refresh` Flag

Introduce a new `--refresh` (or `--force`) CLI flag to `gh aw update` that enables overwrite semantics per-invocation, while preserving the existing add-only default. Users who want to re-sync managed assets would pass the flag explicitly.

This option was not chosen because it places the burden on users to discover and use the flag correctly. The add-only default is the source of the reported bug; making overwrite opt-in would perpetuate the confusing behavior for users who run `update` expecting full synchronization. The extra CLI surface also adds maintenance overhead.

#### Alternative 2: Notify-Only Without Overwriting

Detect stale managed assets during `update` and report which files are out of date (e.g., by comparing content hashes), but leave overwriting to a separate explicit command. This preserves local modifications and avoids silent data loss.

This option was not chosen because it introduces a two-step workflow for a common operation and requires persisting or computing content hashes as part of the update pass. It also does not resolve the underlying issue for users who expect `update` to be the single command that brings the environment current.

### Consequences

#### Positive
- `gh aw update` now reliably propagates upstream changes to package-managed assets (workflows, skills, agents) it owns, without requiring manual file deletion beforehand.
- Ownership tracking (`.github/aw/packages`) prevents `update` from silently overwriting files that belong to a different package, or that a user has customized locally.
- A single shared `updateFrontmatterRepoRefsInContentWithResolver` function reduces code duplication and provides a consistent extension point for future frontmatter ref types beyond `skills` and `plugins`.

#### Negative
- Package-managed files that have not drifted from their installed state are still overwritten during `update` with no diff shown and no interactive confirmation; users who intentionally accept upstream changes without inspecting them will not see a preview.
- Encountering an unowned or drifted destination now surfaces as a hard failure (requiring `--force`) rather than a silent skip, which is a behavior change for scripts/pipelines that previously relied on `update` completing without operator intervention in that case.

#### Neutral
- The `packageAgentDestinationPath` helper function is removed since the destination path is now computed inline at the call site; this is a minor API surface reduction with no behavioral impact.
- Test coverage is added for the plugin ref update path, for the manifest refresh (owned-overwrite) path, and for the refusal to overwrite unowned/drifted destinations, establishing regression baselines for the new semantics.

---

*ADR created by [adr-writer agent] and finalized to reflect the ownership-gated overwrite semantics implemented in this PR.*
