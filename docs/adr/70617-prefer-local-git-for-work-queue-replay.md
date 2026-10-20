# ADR-70617: Prefer Local Git for Work Queue Repository Resolution and Replay

**Date**: 2026-10-20
**Status**: Proposed; maintainer acceptance pending
**Deciders**: pelikhan (PR author); no additional deciders or approval confirmed

---

### Context

Work queue commands (`gh aw work-queue ...`) resolve a repository slug, while the runtime Git ledger client reads and validates queue branch history. Repository resolution previously tried `gh repo view` before local Git; runtime reads repeatedly fetched and checked Git history that had already been validated locally. These operations add latency and network dependencies even when usable local information exists. Delivery verification also needs diagnostics that distinguish unavailable verification, mismatched evidence and exhausted budgets without exposing task content, resource identities, receipts or credentials.

### Decision

Prefer local Git to avoid redundant hosted repository lookups and queue-history fetches. Change the shared, cached `GetCurrentRepoSlug` helper, not just the work queue commands: when `GH_REPO` is unset, resolve a supported GitHub `origin` remote first and use `gh repo view` only if local resolution is unavailable. When `GH_REPO` is set, defer to `gh` for that explicit selection; work queue commands with an explicit `--repo` bypass inference. Offline `replay --file` remains independent of checkout-based repository inference.

For a fork checkout whose `origin` points to the fork and `upstream` points to the parent, inferred commands now select the fork, even if `gh` would select the parent. This deliberately favors the checked-out repository over `gh`'s default selection across all `GetCurrentRepoSlug` callers. Explicit repository selection (`--repo` on commands that support it, or `GH_REPO`) remains the way to target the parent.

In the runtime Git ledger client (`work_queue_git_transport.cjs`), reuse validated local queue history within the transport instance only when a fresh authenticated remote advertisement reports the same head commit. A changed head still triggers fetch, shallow-history rejection and `git fsck` connectivity validation. Invalid or missing ancestry fails closed with no fallback to hosted Git APIs; reuse does not remove the authenticated network check.

Add privacy-preserving delivery diagnostics under the native Go namespaces (`workqueue:delivery`, `workqueue:native_delivery`) and runtime JavaScript namespaces (`work-queue:*`). Only the native exhausted-budget diagnostic becomes always visible; other native delivery details remain `DEBUG` gated. Runtime exhaustion is already always visible and retains that behavior. These delivery diagnostics emit fixed categories, counts and flags rather than sensitive content.

### Alternatives Considered

The following comparisons assess the existing behavior and design options during this ADR review; they do not claim a historical evaluation of every option.

#### Alternative 1: Retain hosted-first repository resolution

Keeping `gh repo view` first preserves `gh`'s default repository selection, including fork-parent selection, but incurs a hosted lookup before trying an available local origin. Restricting origin-first behavior to work queue commands would also preserve other callers' behavior, at the cost of different inference rules across commands. The shared helper change favors consistent local-origin inference and accepts the broader fork-selection tradeoff.

#### Alternative 2: Cache fetched history without rechecking the remote head

Skipping remote checks after validation would avoid more network work, but could miss head changes and loss of repository access. The chosen design retains an authenticated advertisement on each read. Keeping the existing repeated fetch and connectivity checks avoids cache state, but repeats work when the advertised head has not changed.

#### Alternative 3: Fall back to hosted Git APIs when local ancestry is invalid

A hosted fallback might recover from unavailable local objects, but would introduce a second read path with its own validation requirements. This decision retains the Git-only, fail-closed ancestry contract rather than treating a hosted response as recovery from failed Git validation.

### Consequences

#### Positive
- Repository inference avoids a hosted API lookup when a supported GitHub `origin` exists and no explicit override is set; live queue operations still require authenticated remote access.
- Repeated runtime replay within a transport instance avoids redundant fetches and `fsck` passes when the authenticated advertised head is unchanged.
- Operators receive a native exhaustion diagnostic without enabling `DEBUG`, while detailed diagnostics remain opt-in and privacy preserving.

#### Negative
- All shared `GetCurrentRepoSlug` callers can now infer a fork instead of its upstream parent. Users targeting the parent must explicitly select it with a supported `--repo` flag or `GH_REPO`.
- The ledger client holds extra state (`verifiedHead`), and correctness now depends on comparing the advertised head with locally validated history.
- Diagnostic categories and flags must remain free of sensitive data as delivery paths evolve.

#### Neutral
- Error messages from local lookups no longer include the remote URL, which reduces information disclosure.
- Documentation in `docs/src/content/docs/reference/work-queue.md` now describes lookup precedence and the diagnostic namespaces.

---

*The automation-generated draft has been reviewed against the implementation. Proposed status records the decision without asserting maintainer approval.*
