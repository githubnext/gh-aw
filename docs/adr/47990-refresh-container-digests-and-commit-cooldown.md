# ADR-47990: Refresh Container Digests on Update and Enforce Commit-Age Cooldown

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw update` command manages two categories of pinned references: container image digest pins stored in `actions-lock.json`, and workflow source refs (branch, commit SHA, or semantic-version tag). Two independent problems were discovered:

1. **Stale container pins**: Container image tags (e.g., `image:tag`) can move to a new manifest digest after the initial pin is written. The previous implementation treated existing `pinned_image` entries as always current and skipped re-resolving them, leaving lock files referencing outdated digests when upstream images were republished.

2. **Unsafe moving-ref adoption**: Branch-tracked and commit-SHA-tracked workflow sources had no commit-age gate. A freshly pushed commit (seconds or minutes old) could be adopted immediately during an update run, creating operational risk analogous to the release cooldown problem already solved for tag-based sources.

Additionally, lock files compiled with digest-pinned references (`image:tag@sha256:...`) were not being normalized back to their base image ref before cache key lookups, causing reconciliation mismatches.

### Decision

We decided to (a) always re-resolve container digests for images already present in `actions-lock.json` during `gh aw update` runs (controlled by a new `refreshExisting` option on the internal `updateContainerPins` function), and (b) enforce a fixed 3-day commit-age cooldown on branch-tracked and commit-SHA-tracked workflow sources, gated by the same cooldown flag that controls release cooldowns. The commit cooldown is non-configurable (always 3 days) except when the user explicitly disables cooldowns entirely. Digest-pinned lock file image refs are normalized to their base tag before cache reconciliation to eliminate key mismatches.

### Alternatives Considered

#### Alternative 1: Manual Digest Refresh (opt-in flag)

Expose a `--refresh-pins` flag so users can explicitly request container digest re-resolution rather than doing it on every update run. This avoids extra `docker manifest inspect` (or equivalent) calls in normal update flows and gives users explicit control.

Not chosen because the goal of `gh aw update` is to keep the lock file fully synchronized. Silent staleness — where a container digest is out of date but nothing prompts the user to notice — is the worse failure mode. The Docker call is already non-fatal (fails gracefully when Docker is unavailable), so the cost of always refreshing is low.

#### Alternative 2: Time-Based Cache Invalidation for Existing Pins

Instead of always re-resolving, introduce a TTL (e.g., 24 hours) on cached container pins: refresh a pin only if it was last resolved more than TTL ago, storing a timestamp alongside the digest.

Not chosen because it adds persistent timestamp state to `actions-lock.json`, complicating the lock file schema and migration. It also creates a window of guaranteed staleness equal to the TTL, whereas always re-resolving on update provides the freshest state at no additional schema cost.

#### Alternative 3: Per-Ref Commit Cooldown Configured by the User

Allow the commit cooldown to be configured separately from the release cooldown (e.g., `--commit-cooldown 2d`), giving fine-grained control over branch vs. release cadence.

Not chosen because the immediate need is to close a safety gap (too-fresh commits being adopted), and a fixed 3-day constant is already well-understood from the existing release cooldown precedent. Adding a second configurable flag increases UX complexity without clear current demand; this can be revisited if operators need different thresholds.

### Consequences

#### Positive
- Container pins in `actions-lock.json` stay synchronized with upstream image manifest changes across update runs, removing a category of silent staleness.
- Branch-tracked and commit-SHA-tracked workflow sources are subject to the same stability guarantee as tag-based releases: at least 3 days must elapse before a moving ref is adopted.
- Digest-pinned image refs in lock files are normalized before cache key lookup, eliminating reconciliation mismatches between `image:tag@sha256:...` and `image:tag` keys.
- Dependency-injected `containerPinUpdateDeps` and extended `workflowUpdateDeps` structs make both behaviors unit-testable without Docker or live GitHub API calls.

#### Negative
- Every `gh aw update` run now makes one additional `docker manifest inspect` (or equivalent) call per pinned container image, even when the digest has not changed. In environments where Docker is unavailable this is already non-fatal, but it does add latency and noise in verbose output.
- The commit-age cooldown is not user-configurable independently of the release cooldown. Users who want aggressive commit tracking (e.g., CI environments where they own both sides of the workflow repo) cannot lower the 3-day window without disabling all cooldowns.
- The `getLatestBranchCommitSHA` API call was changed to the `/repos/{repo}/commits/{ref}` endpoint (which returns richer JSON) rather than `/repos/{repo}/branches/{branch}` with a `--jq .commit.sha` filter. This increases response payload size and may surface different error shapes for non-existent refs.

#### Neutral
- The public `UpdateContainerPins` function retains its existing signature and defaults (`refreshExisting: false`) for callers outside of `RunUpdateWorkflows`, preserving backward compatibility for any other call sites.
- Branch-commit cache (`branchCommitCache`) now stores `latestBranchCommitInfo` structs (SHA + CommittedAt) instead of plain strings, which is a breaking change to the in-memory cache type — harmless since the cache is process-scoped and cleared at the start of each update run.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
