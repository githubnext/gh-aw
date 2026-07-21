# ADR-47046: Add `actions: write` Permission to Cache-Save Jobs

**Date**: 2026-07-21
**Status**: Draft
**Deciders**: Unknown (Copilot SWE Agent, pelikhan)

---

### Context

GitHub's cache-reservation backend (`actions/cache`) requires the calling token to have at least one writable scope before it will accept a cache-save operation. The `update_cache_memory` job (in `pkg/workflow/cache.go`) and the `conclusion` job (in `pkg/workflow/notify_comment.go`) were generated with `permissions: {}` (empty, i.e. read-only), which caused every cache-save step to fail with `cache write denied: token has no writable scopes`. Because all cache steps carry `continue-on-error: true`, these failures were silent: runs stayed green while `cache-memory` and daily-AIC lineage were never persisted. The fix must grant the minimum additional scope required by the GitHub API without broadening the attack surface of these jobs.

### Decision

We decided to replace `NewPermissionsEmpty()` with `NewPermissionsActionsWrite()` for the `update_cache_memory` job, and to conditionally add `actions: write` to the `conclusionPerms` block when `hasMaxDailyAICGuardrail && WorkflowID != ""` (i.e., exactly when `buildDailyAICUsageCacheSteps` injects an `actions/cache/save` step). In dev mode (local action checkout), `contents: read` is additionally set on the `update_cache_memory` job because the action checkout requires repository access. This grants the minimum privilege required by GitHub's cache backend while leaving all other permission surfaces unchanged.

### Alternatives Considered

#### Alternative 1: Keep Empty Permissions and Use a Repository-Level Token

Rely on a separately injected PAT or app token that already has `actions: write` scope, rather than elevating the GITHUB_TOKEN scope on these jobs. This avoids any permission change to generated lock files.

Rejected because: it introduces an external secret dependency and a secret-rotation burden for a routine cache operation. The GITHUB_TOKEN with `actions: write` is the idiomatic GitHub Actions approach for cache saves and does not require any additional secrets infrastructure.

#### Alternative 2: Grant `contents: write` Instead of `actions: write`

Elevate to `contents: write`, which also satisfies the cache backend's "needs a writable scope" check.

Rejected because: `contents: write` grants write access to repository contents, which is far beyond what a cache-save step needs. `actions: write` is the narrowest scope that satisfies the requirement and follows the least-privilege principle documented in GitHub's Actions security hardening guide.

### Consequences

#### Positive
- Cache saves for `update_cache_memory` and daily-AIC `conclusion` jobs now succeed, so `cache-memory` state and daily-AIC lineage are properly persisted across runs.
- The fix uses the minimum necessary privilege (`actions: write` only), preserving the least-privilege posture of all affected jobs.
- A conditional logic path ensures that `actions: write` is added to `conclusionPerms` only when the cache-save step is actually injected, preventing unnecessary permission grants in other conclusion job variants.

#### Negative
- All 260 compiled lock files required regeneration, producing a large diff that makes the core logic change harder to review at first glance.
- Any future refactor of `NewPermissionsEmpty()` usage must audit whether the call site performs cache writes, to avoid reintroducing silent failures.

#### Neutral
- The dev-mode code path (`setupActionRef != "" && len(c.generateCheckoutActionsFolder(data)) > 0`) additionally receives `contents: read` on the `update_cache_memory` job; this is consistent with existing dev-mode checkout requirements and does not affect production workflows.
- The `continue-on-error: true` pattern on cache steps remains unchanged; the fix resolves the root cause rather than removing the error-suppression mechanism.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
