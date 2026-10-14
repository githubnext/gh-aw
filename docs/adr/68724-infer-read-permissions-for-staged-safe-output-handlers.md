# ADR-68724: Infer Read Permissions for Staged Safe-Output Handlers

**Date**: 2026-10-14
**Status**: Proposed
**Deciders**: PR #68724 maintainers

---

### Context

Safe-output handlers that support `required-labels` and `required-title-prefix` fetch their target (for example via `github.rest.issues.get`) to evaluate the filter *before* they check whether the run is staged. Until now, permission inference treated staged handlers as needing no permissions at all, so a fully staged configuration compiled the `safe_outputs` job with `permissions: {}`. On private repositories those pre-preview reads then failed with `403 Resource not accessible by integration`, so a preview-only run crashed (issue #68480). A one-off fix had already been applied for `add-comment` (#66571), but every other handler that reads its target before the staged short-circuit remained broken. The constraint is that staged runs must keep the strong guarantee of never receiving write scopes while still being able to perform the reads they actually execute.

### Decision

We will compute staged-handler permissions from the reads a handler actually performs instead of assuming staged means "no permissions". A new `computeSafeOutputHandlerPermissions` function wraps each handler's `PermissionBuilder`: when the builder returns `nil` (staged) but the handler is configured, it grants a handler-specific read-only scope set — `issues: read`, `pull-requests: read`, `discussions: read`, `organization-projects: read`, or the broader read set for `merge-pull-request` — plus `issues: read` when supported `required-labels` or `required-title-prefix` filters are configured. The same function is used both for job-level `permissions:` and for global and per-handler GitHub App token scoping, so job permissions and minted token scopes stay consistent. Organization-project scopes apply only to App tokens, not the Actions job permission block. Handlers that need no reads still yield `nil`, preserving explicit `permissions: {}` rendering.

Discussion creation, project status updates, review operations, and comment hiding receive the read scopes required to resolve their targets before previewing. `add-comment` receives Issues read access only when issue or pull-request targets are enabled; discussion-only comments do not use Issues filters. `set-issue-type` serializes both supported filters into its runtime configuration. `hide-comment` does not implement label/title filters, so its unused filter configuration embed is removed rather than advertising unsupported safety gates.

### Alternatives Considered

#### Alternative 1: Skip or soften the target fetch in staged handlers

Change each handler script (for example `remove_labels.cjs`) to short-circuit on `staged` before performing the filter lookup, so no extra permissions are needed. This keeps `permissions: {}` for staged runs, which is the strongest security posture. It was rejected because the preview would then no longer reflect whether the filter would have accepted the target — staged mode would report outputs that a live run would reject, defeating the purpose of the preview. It also requires parallel edits in every handler script with no compiler-level guarantee of consistency.

#### Alternative 2: Continue per-handler ad-hoc fixes (extend the `add-comment` special case)

Keep the existing shape where specific handlers get hard-coded read grants, adding `remove-labels` and others as bugs are reported. This matches the existing precedent from #66571 but is reactive: each newly filtered handler reintroduces the same 403 class of failure, and the special cases were not shared with GitHub App token scoping, so app-token runs could still diverge from job permissions.

### Consequences

#### Positive
- Staged runs on private repositories no longer fail with 403 when `required-labels` or `required-title-prefix` is configured, and previews now reflect the real filter outcome.
- Job permissions and GitHub App token scopes are derived from one shared function, eliminating drift between the two code paths.
- Staged handlers still never receive write scopes; the change is strictly read-only and regression tests assert `: write` is absent from the compiled staged job.

#### Negative
- Staged jobs that previously compiled to `permissions: {}` now carry read scopes, slightly widening the token surface of preview-only runs.
- The read-scope mapping is a hand-maintained `switch` over handler keys, so a new handler that reads its target before previewing must be added manually or it will regress.
- The filter detection uses reflection on `RequiredLabels` / `RequiredTitlePrefix` fields, which is not checked by the compiler and can silently stop matching if those fields are renamed.

#### Neutral
- Existing lock files change: `.github/workflows/poem-bot.lock.yml` gains `pull-requests: read` and must be regenerated along with other affected workflows.
- Documentation in `docs/src/content/docs/reference/safe-outputs.md` now states explicitly that staged handlers may read target data.
- Some grants are coarser than strictly required (for example `issues: read` for filter evaluation on pull-request handlers, since PRs are issues in the REST API).

---

*Maintainer acceptance remains pending; this proposal documents the implemented decision without claiming approval.*
