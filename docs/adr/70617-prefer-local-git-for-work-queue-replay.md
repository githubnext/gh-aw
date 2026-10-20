# ADR-70617: Prefer Local Git for Work Queue Repository Resolution and Replay

**Date**: 2026-05-13
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify other reviewers]

---

### Context

Work queue commands (`gh aw work-queue ...`) and the runtime Git ledger client need two things that were previously obtained from hosted GitHub lookups: the current repository slug, which came from `gh repo view` first, and the queue branch history, which was re-fetched and re-validated on every runtime read. Hosted lookups add latency and network dependencies. They can also fail in environments where local Git already has the needed information. Repeated replay refetched history that had already been validated locally. Delivery verification was also opaque: operators saw completed work items with no verified Result and had no clear way to find out why. Diagnostics must not leak task content, resource identities, receipts, or credentials.

### Decision

We will resolve the repository from the local `origin` Git remote first. `gh repo view` becomes the fallback. Explicit `--repo` and `GH_REPO` keep precedence, and offline `replay --file` does not require a checkout. In the runtime Git ledger client (`work_queue_git_transport.cjs`), we will reuse validated local queue history when a fresh authenticated remote advertisement reports the same head commit. A changed head still triggers fetch, shallow-history rejection and `git fsck` connectivity validation. Invalid or missing ancestry fails closed with no fallback to hosted Git APIs. We will also add privacy-preserving, namespace-gated debug logging (`workqueue:delivery`, `workqueue:native_delivery`, `work-queue:*`) and an always-visible exhaustion summary. These outputs emit only fixed categories, counts and flags.

### Alternatives Considered

#### Alternative 1: Keep hosted-first lookups (`gh repo view` first, always refetch)

This was the existing behavior. It is simple, and the hosted API is authoritative about the canonical repository. It was rejected because it makes every queue inspection and replay depend on the network and on API availability, and it repeats work already validated locally. The PR states that local Git information should be used before hosted lookups.

#### Alternative 2: Cache fetched history without rechecking the remote head

Skipping the fetch entirely once history is validated would be cheaper still. It was rejected because it cannot detect head changes or revoked access. The chosen design still performs an authenticated advertisement on each read, so access is rechecked and new heads are fetched and validated.

#### Alternative 3: Fall back to hosted Git APIs when local ancestry is invalid

This would recover from corrupt or shallow local history automatically. It was rejected because the queue's ledger integrity relies on validated Git ancestry. Silently switching to a different source of truth would weaken fail-closed guarantees.

### Consequences

#### Positive
- Queue commands work and resolve the repository without a hosted API call when a GitHub `origin` remote exists.
- Repeated runtime replay avoids redundant fetches and `fsck` passes when the queue head is unchanged.
- Operators get actionable, categorized delivery-verification diagnostics and an exhaustion summary, with no sensitive content logged.

#### Negative
- `origin` may point to a fork or a different repository than `gh`'s default selection would. Users must set `--repo` or `GH_REPO` to override it, which is a behavior change for `getCurrentRepoSlug` callers across the CLI.
- The ledger client holds extra state (`verifiedHead`), and correctness now depends on comparing the advertised head with locally validated history.
- Many new log points across the delivery paths must be kept free of sensitive data as the code evolves.

#### Neutral
- Error messages from local lookups no longer include the remote URL, which reduces information disclosure.
- Documentation in `docs/src/content/docs/reference/work-queue.md` now describes lookup precedence and the diagnostic namespaces.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
