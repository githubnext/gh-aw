# ADR-70617: Prefer Local Git for Work Queue Repository Resolution and Replay

**Date**: 2026-10-20
**Status**: Proposed; maintainer acceptance pending
**Deciders**: pelikhan (PR author); no additional deciders or approval confirmed

---

### Context

Work queue commands (`gh aw work-queue ...`) resolve a repository slug, while the runtime Git ledger client reads and validates queue branch history. Repository resolution previously tried `gh repo view` before local Git; runtime reads repeatedly fetched and checked Git history that had already been validated locally. These operations add latency and network dependencies even when usable local information exists. The shared resolver also serves workflow runs and secret uploads, which must retain `gh`'s selected, canonical repository rather than silently switch from a configured parent to a checkout fork. Delivery verification also needs diagnostics that distinguish unavailable verification, mismatched evidence and exhausted budgets without exposing task content, resource identities, receipts or credentials.

### Decision

Prefer local Git for queue-specific repository discovery and avoid redundant queue-history fetches. A dedicated queue resolver tries a supported GitHub `origin` first when `GH_REPO` is unset, then falls back to the shared resolver when local information is unavailable. The shared, cached `GetCurrentRepoSlug` retains hosted-first canonical `gh repo view` selection for unrelated commands, with local Git as its fallback. When `GH_REPO` is set, defer to `gh` for that explicit selection and fail rather than silently target the checkout if that selection cannot be resolved. Work queue commands with an explicit `--repo` bypass inference. Offline `replay --file` remains independent of checkout-based repository inference.

For a fork checkout whose `origin` points to the fork and `upstream` points to the parent, inferred queue commands select the fork, even if `gh` is configured to select the parent. Workflow runs, secret uploads and other shared-resolver callers retain the parent selection. Explicit repository selection (`--repo` or `GH_REPO`) remains the way to target the parent for queues. This queue-scoped decision supersedes earlier descriptions of a global origin-first change.

Local discovery is not proof of canonical hosted identity: a stale origin after a repository rename can still identify the old slug. Queue authentication and ledger scope checks remain fail-closed; this decision does not authorize rewriting historical ledger identity or switching repositories after an authentication failure. An explicit canonical `--repo` or `GH_REPO` avoids stale checkout inference. Shared-resolver callers obtain canonical rename resolution from `gh` without changing their targeting semantics.

In the runtime Git ledger client (`work_queue_git_transport.cjs`), reuse validated local queue history within the transport instance only when a fresh authenticated remote advertisement reports the same head commit. A changed head still triggers fetch, shallow-history rejection and `git fsck` connectivity validation. Invalid or missing ancestry fails closed with no fallback to hosted Git APIs; reuse does not remove the authenticated network check.

Add privacy-preserving delivery diagnostics under the native Go namespaces (`workqueue:delivery`, `workqueue:native_delivery`) and runtime JavaScript namespaces (`work-queue:*`). Only the native exhausted-budget diagnostic becomes always visible; other native delivery details remain `DEBUG` gated. Runtime exhaustion is already always visible and retains that behavior. These delivery diagnostics emit fixed categories, counts and flags rather than sensitive content.

### Alternatives Considered

The following comparisons assess the existing behavior and design options during this ADR review; they do not claim a historical evaluation of every option.

#### Alternative 1: Retain hosted-first repository resolution

Keeping `gh repo view` first for every queue command preserves `gh`'s default repository selection, including fork-parent selection, but incurs a hosted lookup before trying an available local origin. Changing the shared helper to origin-first would avoid that lookup across commands, but could split workflow-run targets from subsequent polling and change secret-upload targets. The chosen queue-only helper accepts different discovery rules across commands to preserve those existing callers.

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
- Queue commands can infer a fork instead of its upstream parent. Users targeting the parent must explicitly select it with `--repo` or `GH_REPO`; other commands retain `gh`'s selected default.
- A valid-looking local origin cannot establish canonical hosted identity offline. Stale renamed origins require an explicit canonical queue target; authentication and ledger-scope failures do not automatically retarget a command.
- The ledger client holds extra state (`verifiedHead`), and correctness now depends on comparing the advertised head with locally validated history.
- Diagnostic categories and flags must remain free of sensitive data as delivery paths evolve.

#### Neutral
- Shared URL-parser diagnostics and local lookup errors omit raw URLs and extracted unvalidated paths, including under CLI debug and runner debug, to avoid exposing embedded credentials.
- Documentation in `docs/src/content/docs/reference/work-queue.md` now describes lookup precedence and the diagnostic namespaces.

---

*The automation-generated draft has been reviewed against the implementation. Proposed status records the decision without asserting maintainer approval.*
