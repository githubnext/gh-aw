# ADR-47730: Cooldown Fallback to Older Release

**Date**: 2026-07-24
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw update` command applies a cooldown policy to avoid pinning actions and
workflow sources to releases that are too recent (e.g. within 7 days of publication),
providing a safety buffer against newly-discovered bugs. Before this change, when the
newest available release was still within the cooldown window the update was silently
skipped entirely, leaving users on the current (potentially outdated) version even when
an earlier—but still upgrading—release had already cooled down. This affected both the
`updateActions` path (lock-file pins) and the `updateActionRefsInContentWithDeps` path
(inline workflow refs), as well as the `resolveLatestReleaseWithDeps` helper used for
workflow sources.

### Decision

We will change the cooldown handling in all three update paths so that, when the newest
upgrade candidate is still in cooldown, the system fetches the full release list, sorts
upgrade candidates newest-first, and walks them until it finds the first release that has
passed the cooldown period. That cooled-down release is used instead of skipping the
update. If every candidate is still hot, the behaviour degrades gracefully to the previous
no-op (return `currentRef` / skip). We also guard `ActionCache.Set` against empty SHAs to
prevent an unresolvable pin from being written to `actions-lock.json`.

### Alternatives Considered

#### Alternative 1: Keep the original skip-on-cooldown behaviour

The prior approach—skip the entire update when the newest release is in cooldown—was
simple: one cooldown check per repo, no additional API calls, predictable behaviour. It
was not chosen because it left users on older versions when a perfectly valid, cooled-down
intermediate release existed, defeating the point of running the update command.

#### Alternative 2: Disable or make cooldown opt-in

Removing the cooldown constraint entirely would eliminate the fallback complexity and
guarantee every update reaches the absolute latest release. This was not chosen because
the cooldown policy exists specifically to avoid pinning to too-fresh releases that may
carry undiscovered bugs. Removing it would undermine that safety goal.

#### Alternative 3: Surface a warning and require manual override

Instead of automatically selecting an older release, the tool could report that the latest
is in cooldown and require the user to pass `--ignore-cooldown` or similar to proceed.
This was not chosen because it places operational burden on the user for a decision (use
the latest cooled-down release) that is both safe and predictable.

### Consequences

#### Positive
- Users automatically receive the latest upgrade that has passed the cooldown period rather than being silently held back.
- `checkCoolDown` is now an injectable dependency on both `actionUpdateDeps` and `workflowUpdateDeps`, enabling deterministic unit testing of cooldown scenarios without time manipulation.
- The `ActionCache.Set` empty-SHA guard prevents `actions-lock.json` from being polluted with unresolvable pin entries.

#### Negative
- Each cooldown fallback triggers an additional GitHub Releases API call (`/repos/{owner}/{repo}/releases`) to enumerate candidates; this increases latency and API usage for repos whose latest release is in cooldown.
- The release-selection logic is more complex (collect, sort descending, iterate with per-candidate cooldown checks) compared to the previous single-check approach, increasing maintenance surface.

#### Neutral
- The fallback is fail-open: if the releases API call fails or returns no suitable candidates, the code falls back to the original skip behaviour, so existing semantics are preserved on error.
- All three update paths (`updateActions`, `updateActionRefsInContentWithDeps`, `resolveLatestReleaseWithDeps`) now share the same logical pattern, improving consistency at the cost of the change touching multiple callsites.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
